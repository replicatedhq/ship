package apptype

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-getter"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helm"
	helm2 "github.com/replicatedhq/ship/pkg/lifecycle/render/helm"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Inspector interface {
	// DetermineApplicationType loads and application from upstream,
	// returning the app type and the local path where its been downloaded (when applicable),
	DetermineApplicationType(
		ctx context.Context,
		upstream string,
	) (appType string, localPath string, err error)
}

func NewInspector(
	logger log.Logger,
	gh *githubclient.GithubClient,
	fs afero.Afero,
	v *viper.Viper,
	stateManager state.Manager,
	commands helm2.Commands,
	ui cli.Ui,
) Inspector {
	return &inspector{
		logger: logger,
		github: gh,
		fs:     fs,
		viper:  v,
		state:  stateManager,
		helm:   commands,
		ui:     ui,
	}
}

type inspector struct {
	logger log.Logger
	github *githubclient.GithubClient
	fs     afero.Afero
	viper  *viper.Viper
	state  state.Manager
	helm   helm2.Commands
	ui     cli.Ui
}

func isGoGettable(path string) bool {
	_, err := getter.Detect(path, "", getter.Detectors)
	if err != nil {
		return false
	}
	return true
}

var githubTreeRegex = regexp.MustCompile(`^[htps:/]*[w.]*github\.com/([^/]+)/([^/]+)/tree/([^/]+)/?(.*)`)

// if this path is a github path of the form `github.com/OWNER/REPO/tree/REF/SUBDIR`, change it to the go-getter form
// of `github.com/OWNER/REPO?ref=REF//SUBDIR`
// otherwise return the unmodified path
func untreeGithub(path string) string {
	matches := githubTreeRegex.FindStringSubmatch(path)
	if matches == nil || len(matches) < 5 {
		return path
	}

	owner := matches[1]
	repo := matches[2]
	ref := matches[3]
	subdir := matches[4]

	return fmt.Sprintf("github.com/%s/%s?ref=%s//%s", owner, repo, ref, subdir)
}

func (r *inspector) DetermineApplicationType(
	ctx context.Context,
	upstream string,
) (appType string, localPath string, err error) {
	debug := level.Debug(log.With(r.logger, "method", "determineApplicationType"))

	// hack hack hack
	isReplicatedApp := strings.HasPrefix(upstream, "replicated.app") ||
		strings.HasPrefix(upstream, "staging.replicated.app") ||
		strings.HasPrefix(upstream, "local.replicated.app")
	if isReplicatedApp {
		return "replicated.app", "", nil
	}

	upstream = untreeGithub(upstream)
	if isGoGettable(upstream) {
		// get with go-getter
		return r.determineTypeFromContents(ctx, upstream)
	}

	// otherwise we're fetching the chart with `helm fetch`
	chartRepoURL := r.viper.GetString("chart-repo-url")
	chartVersion := r.viper.GetString("chart-version")
	// persist helm options
	err = r.state.SaveHelmOpts(chartRepoURL, chartVersion)
	if err != nil {
		return "", "", errors.Wrap(err, "write helm opts")
	}

	debug.Log("event", "helm.init")
	err = r.helm.Init()
	if err != nil {
		return "", "", errors.Wrapf(err, "helm init")
	}

	debug.Log("event", "helm.fetch")
	localPath, err = r.fetchUnpackChartWithLibHelm(
		upstream,
		chartRepoURL,
		chartVersion,
		constants.InternalTempHelmHome,
	)
	if err != nil {
		return "", "", errors.Wrapf(err, "fetch chart")
	}
	return "helm", localPath, nil
}

func (r *inspector) determineTypeFromContents(
	ctx context.Context,
	upstream string,
) (
	applicationType string,
	checkoutPath string,
	err error,
) {
	debug := level.Debug(r.logger)
	savePath := path.Join(constants.ShipPathInternalTmp, "tmp-repo")
	err = getter.GetAny(savePath, upstream)
	if err != nil {
		return "", "", errors.Wrap(err, "fetch contents with go-getter")
	}
	// if there's a Chart.yaml, assume its a chart
	isChart, err := r.fs.Exists(path.Join(savePath, "Chart.yaml"))
	if err != nil {
		return "", "", errors.Wrap(err, "check for Chart.yaml")
	}
	debug.Log("event", "isChart.check", "isChart", isChart)

	if isChart {
		if err != nil {
			return "", "", errors.Wrapf(err, "copy %s to chart/", savePath)
		}
		return "helm", savePath, nil
	}

	return "k8s", savePath, nil
}

// fetchUnpackChartWithLibHelm fetches and unpacks the chart into a temp directory, then copies the contents of the chart folder to
// the destination dir.
// TODO figure out how to copy files from host into afero filesystem for testing, or how to force helm to fetch into afero
func (r *inspector) fetchUnpackChartWithLibHelm(
	chartRef,
	repoURL,
	version,
	home string,
) (localPath string, err error) {
	debug := level.Debug(log.With(r.logger, "method", "fetchUnpackChartWithLibHelm"))

	debug.Log("event", "helm.unpack")
	tmpDest, err := r.fs.TempDir(constants.ShipPathInternalTmp, "helm-fetch-unpack")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory to unpack to")
	}

	// TODO: figure out how to get files into aferoFs here
	debug.Log("event", "helm.fetch")
	helmOutput, err := helm.Fetch(chartRef, repoURL, version, tmpDest, home)
	if err != nil {
		return "", errors.Wrapf(err, "helm fetch: %s", helmOutput)
	}

	subdir, err := util.FindOnlySubdir(tmpDest, r.fs)
	if err != nil {
		return "", errors.Wrap(err, "find chart subdir")
	}

	return subdir, nil
}
