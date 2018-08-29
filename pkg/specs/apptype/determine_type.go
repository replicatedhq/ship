package apptype

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/gogetter"
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

type FileFetcher interface {
	GetFiles(ctx context.Context, upstream, savePath string) error
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

	// TODO implement a way to choose which github method should be used

	// use the integrated github client if the url is a github url and does not contain "//"
	if util.IsGithubURL(upstream) {
		githubClient := githubclient.NewGithubClient(r.fs, r.logger)
		return r.determineTypeFromContents(ctx, upstream, githubClient)
	}

	upstream = gogetter.UntreeGithub(upstream)
	if gogetter.IsGoGettable(upstream) {
		// get with go-getter
		fetcher := gogetter.GoGetter{Logger: r.logger, FS: r.fs}
		return r.determineTypeFromContents(ctx, upstream, &fetcher)
	}

	return "", "", errors.New(fmt.Sprintf("upstream %s is not a replicated app, a github repo, or compatible with go-getter", upstream))
}

func (r *inspector) determineTypeFromContents(
	ctx context.Context,
	upstream string,
	fetcher FileFetcher,
) (
	applicationType string,
	checkoutPath string,
	err error,
) {
	debug := level.Debug(r.logger)
	savePath := path.Join(constants.ShipPathInternalTmp, "tmp-repo")

	err = fetcher.GetFiles(ctx, upstream, savePath)
	if err != nil {
		return "", "", err
	}

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
