package helm

import (
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
)

// ChartFetcher fetches a chart based on an asset. it returns
// the location that the chart was unpacked to, usually a temporary directory
type ChartFetcher interface {
	FetchChart(asset api.HelmAsset, meta api.ReleaseMetadata) (string, error)
}

// ClientFetcher is a ChartFetcher that does all the pulling/cloning client side
type ClientFetcher struct {
	Logger log.Logger
}

func (f *ClientFetcher) FetchChart(asset api.HelmAsset, meta api.ReleaseMetadata) (string, error) {
	debug := log.With(level.Debug(f.Logger), "fetcher", "client")

	if asset.Local != nil {
		debug.Log("event", "chart.fetch", "source", "local", "root", asset.Local.ChartRoot)
		// this is not great but it'll do -- prepend `installer` since this is off of inline assets
		chartRootPath := path.Join(constants.InstallerPrefix, asset.Local.ChartRoot)

		return chartRootPath, nil
	}

	debug.Log("event", "chart.fetch.fail", "reason", "unsupported")
	return "", errors.New("only 'local' chart rendering is supported")
}

// NewFetcher makes a new chart fetcher
func NewFetcher(logger log.Logger) ChartFetcher {
	return &ClientFetcher{
		Logger: logger,
	}
}
