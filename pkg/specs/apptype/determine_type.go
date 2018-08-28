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
	"github.com/replicatedhq/ship/pkg/state"
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
	fs afero.Afero,
	v *viper.Viper,
	stateManager state.Manager,
	ui cli.Ui,
) Inspector {
	return &inspector{
		logger: logger,
		fs:     fs,
		viper:  v,
		state:  stateManager,
		ui:     ui,
	}
}

type inspector struct {
	logger log.Logger
	fs     afero.Afero
	viper  *viper.Viper
	state  state.Manager
	ui     cli.Ui
}

func isGoGettable(path string) bool {
	_, err := getter.Detect(path, "", getter.Detectors)
	if err != nil {
		return false
	}
	return true
}

var githubTreeRegex = regexp.MustCompile(`^[htps:/]*[w.]*github\.com/([^/?=]+)/([^/?=]+)/tree/([^/?=]+)/?(.*)$`)
var githubRegex = regexp.MustCompile(`^[htps:/]*[w.]*github\.com/([^/?=]+)/([^/?=]+)(/(.*))?$`)

// if this path is a github path of the form `github.com/OWNER/REPO/tree/REF/SUBDIR` or `github.com/OWNER/REPO/SUBDIR`,
// change it to the go-getter form of `github.com/OWNER/REPO?ref=REF//SUBDIR` with a default ref of master
// otherwise return the unmodified path
func untreeGithub(path string) string {
	var owner, repo, ref, subdir string

	matches := githubTreeRegex.FindStringSubmatch(path)
	if matches != nil && len(matches) == 5 {
		owner = matches[1]
		repo = matches[2]
		ref = matches[3]
		subdir = matches[4]
	} else if matches = githubRegex.FindStringSubmatch(path); matches != nil && len(matches) == 5 {
		owner = matches[1]
		repo = matches[2]
		ref = "master"
		subdir = matches[4]
	}

	if owner != "" {
		return fmt.Sprintf("github.com/%s/%s?ref=%s//%s", owner, repo, ref, subdir)
	}

	return path
}

func (r *inspector) DetermineApplicationType(
	ctx context.Context,
	upstream string,
) (appType string, localPath string, err error) {

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

	return "", "", errors.New(fmt.Sprintf("upstream %s is not compatible with go-getter", upstream))
}

// TODO figure out how to copy files from host into afero filesystem for testing, or how to force go-getter to fetch into afero
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

	// if there is a `.git` directory, remove it - it's dynamic and will break the content hash used by `ship update`
	gitPresent, err := r.fs.Exists(path.Join(savePath, ".git"))
	if err != nil {
		return "", "", errors.Wrap(err, "check for .git directory")
	}
	if gitPresent {
		err := r.fs.RemoveAll(path.Join(savePath, ".git"))
		if err != nil {
			return "", "", errors.Wrap(err, "remove .git directory")
		}
	}
	debug.Log("event", "gitPresent.check", "gitPresent", gitPresent)

	// if there's a Chart.yaml, assume its a chart
	isChart, err := r.fs.Exists(path.Join(savePath, "Chart.yaml"))
	if err != nil {
		return "", "", errors.Wrap(err, "check for Chart.yaml")
	}
	debug.Log("event", "isChart.check", "isChart", isChart)

	if isChart {
		return "helm", savePath, nil
	}

	return "k8s", savePath, nil
}
