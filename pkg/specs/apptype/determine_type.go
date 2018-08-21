package apptype

import (
	"context"
	"path"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

	// "sure"
	isGithub := strings.HasPrefix(strings.TrimLeft(upstream, "htps:/"), "github.com/")

	if isGithub {
		return r.determineTypeFromGithubContents(ctx, upstream)
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
	helmCmdOutput, err := r.fetchUnpackChartWithLibHelm(
		upstream,
		chartRepoURL,
		chartVersion,
		"chart",
		constants.InternalTempHelmHome,
	)
	if err != nil {
		return "", "", errors.Wrapf(err, "fetch chart with helm: %s", helmCmdOutput)
	}
	return "helm", "chart", nil
}

func (r *inspector) determineTypeFromGithubContents(
	ctx context.Context,
	upstream string,
) (
	applicationType string,
	checkoutPath string,
	err error,
) {
	debug := level.Debug(r.logger)
	savePath := path.Join(constants.ShipPathInternal, "tmp-repo")
	err = r.github.GetRepoContent(ctx, upstream, savePath)
	if err != nil {
		return "", "", errors.Wrap(err, "fetch repo contents")
	}
	defer r.fs.RemoveAll(savePath)
	// if there's a Chart.yaml, assume its a chart
	isChart, err := r.fs.Exists(path.Join(savePath, "Chart.yaml"))
	if err != nil {
		return "", "", errors.Wrap(err, "check for Chart.yaml")
	}

	if isChart {
		destination := constants.HelmChartPath
		err := util.BackupIfPresent(r.fs, destination, debug, r.ui)
		if err != nil {
			return "", "", errors.Wrapf(err, "try backup %s", destination)
		}
		err = r.fs.Rename(savePath, destination)
		if err != nil {
			return "", "", errors.Wrapf(err, "copy %s to chart/", savePath)
		}
		return "helm", destination, nil
	}

	util.BackupIfPresent(r.fs, constants.KustomizeBasePath, debug, r.ui)
	err = r.fs.Rename(savePath, constants.KustomizeBasePath)
	if err != nil {
		return "", "", errors.Wrapf(err, "copy %s to k8s/", savePath)
	}
	return "k8s", constants.KustomizeBasePath, nil
}

// fetchUnpackChartWithLibHelm fetches and unpacks the chart into a temp directory, then copies the contents of the chart folder to
// the destination dir.
// TODO figure out how to copy files from host into afero filesystem for testing, or how to force helm to fetch into afero
func (r *inspector) fetchUnpackChartWithLibHelm(
	chartRef,
	repoURL,
	version,
	dest,
	home string,
) (helmOutput string, err error) {
	debug := level.Debug(log.With(r.logger, "method", "fetchUnpackChartWithLibHelm"))

	err = r.fs.MkdirAll(constants.ShipPathInternal, 0775)
	if err != nil {
		return "", errors.Wrap(err, "unable to create ship directory")
	}

	tmpDest, err := r.fs.TempDir(constants.ShipPathInternal, "helm-fetch-unpack")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory to unpack to")
	}
	defer r.fs.RemoveAll(tmpDest)

	// TODO: figure out how to get files into aferoFs here
	helmOutput, err = helm.Fetch(chartRef, repoURL, version, tmpDest, home)
	if err != nil {
		return helmOutput, err
	}

	subdir, err := util.FindOnlySubdir(tmpDest, r.fs)
	if err != nil {
		return "", errors.Wrap(err, "find chart subdir")
	}

	// check if the destination directory exists - if it does, remove it
	debug.Log("event", "checkExists", "path", dest)
	saveDirExists, err := r.fs.Exists(dest)
	if err != nil {
		return "", errors.Wrapf(err, "check %s exists", dest)
	}

	if saveDirExists {
		debug.Log("event", "removeAll", "path", dest)
		err := r.fs.RemoveAll(dest)
		if err != nil {
			return "", errors.Wrapf(err, "remove %s", dest)
		}
	}

	// rename that folder to move it to the destination directory
	err = r.fs.Rename(subdir, dest)
	if err != nil {
		return "", errors.Wrapf(err, "rename %s to %s", subdir, dest)
	}

	return "", nil
}
