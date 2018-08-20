package helm

import (
	"context"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
)

// ChartFetcher fetches a chart based on an asset. it returns
// the location that the chart was unpacked to, usually a temporary directory
type ChartFetcher interface {
	FetchChart(
		ctx context.Context,
		rootFs root.Fs,
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
		configGroups []libyaml.ConfigGroup,
		templateContext map[string]interface{},
	) (string, error)
}

// ClientFetcher is a ChartFetcher that does all the pulling/cloning client side
type ClientFetcher struct {
	Logger       log.Logger
	GitHub       github.Renderer
	FS           afero.Afero
	HelmCommands Commands
}

func (f *ClientFetcher) FetchChart(
	ctx context.Context,
	rootFs root.Fs,
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) (string, error) {
	debug := log.With(level.Debug(f.Logger), "fetcher", "client")

	if asset.Local != nil {
		debug.Log("event", "chart.fetch", "source", "local", "root", asset.Local.ChartRoot)
		return asset.Local.ChartRoot, nil
	} else if asset.GitHub != nil {
		checkoutDir, err := f.FS.TempDir("", "helmchart")
		if err != nil {
			return "", errors.Wrap(err, "get chart checkout tmpdir")
		}
		asset.GitHub.Dest = checkoutDir
		err = f.GitHub.Execute(
			rootFs,
			*asset.GitHub,
			configGroups,
			meta,
			templateContext,
		)(ctx)

		if err != nil {
			return "", errors.Wrap(err, "fetch github asset")
		}

		return path.Join(checkoutDir, asset.GitHub.Path), nil
	} else if asset.HelmFetch != nil {
		checkoutDir, err := f.FS.TempDir("", "helmchart")
		if err != nil {
			return "", errors.Wrap(err, "get chart checkout tmpdir")
		}

		err = f.HelmCommands.Init()
		if err != nil {
			return "", errors.Wrap(err, "init helm")
		}

		err = f.HelmCommands.Fetch(
			asset.HelmFetch.ChartRef,
			asset.HelmFetch.RepoURL,
			asset.HelmFetch.Version,
			checkoutDir,
			constants.InternalTempHelmHome,
		)
		if err != nil {
			return "", errors.Wrap(err, "helm fetch")
		}

		// find the path that the chart was fetched to
		chartDir, err := util.FindOnlySubdir(checkoutDir, f.FS)
		if err != nil {
			return "", errors.Wrap(err, "failed to find chart dir")
		}

		return chartDir, nil
	}

	debug.Log("event", "chart.fetch.fail", "reason", "unsupported")
	return "", errors.New("only 'local' and 'github' chart rendering is supported")
}

// NewFetcher makes a new chart fetcher
func NewFetcher(
	logger log.Logger,
	github github.Renderer,
	fs afero.Afero,
	helmCommands Commands,
) ChartFetcher {
	return &ClientFetcher{
		Logger:       logger,
		GitHub:       github,
		FS:           fs,
		HelmCommands: helmCommands,
	}
}
