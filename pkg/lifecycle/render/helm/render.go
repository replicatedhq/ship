package helm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
	) func(ctx context.Context) error
}

var _ Renderer = &LocalRenderer{}

// LocalRenderer can add a helm step to the plan, the step will fetch the
// chart to a temporary location and then run a local operation to run the helm templating
type LocalRenderer struct {
	Templater Templater
	Fetcher   ChartFetcher
}

// NewRenderer makes a new renderer
func NewRenderer(cloner ChartFetcher, templater Templater) Renderer {
	return &LocalRenderer{
		Fetcher:   cloner,
		Templater: templater,
	}
}

func (r *LocalRenderer) Execute(
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		chartLocation, err := r.Fetcher.FetchChart(asset, meta)
		if err != nil {
			return errors.Wrap(err, "fetch chart")
		}

		err = r.Templater.Template(chartLocation, asset, meta)
		if err != nil {
			return errors.Wrap(err, "execute templating")
		}
		return nil
	}

}
