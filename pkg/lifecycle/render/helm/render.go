package helm

import (
	"context"

	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"

	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

var _ Renderer = &LocalRenderer{}

// LocalRenderer can add a helm step to the plan, the step will fetch the
// chart to a temporary location and then run a local operation to run the helm templating
type LocalRenderer struct {
	Templater Templater
	Fetcher   ChartFetcher
	GitHub    github.Renderer
}

// NewRenderer makes a new renderer
func NewRenderer(cloner ChartFetcher, templater Templater, github github.Renderer) Renderer {
	return &LocalRenderer{
		Fetcher:   cloner,
		Templater: templater,
		GitHub:    github,
	}
}

func (r *LocalRenderer) Execute(
	rootFs root.Fs,
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		chartLocation, err := r.Fetcher.FetchChart(
			ctx,
			asset,
			meta,
			configGroups,
			templateContext,
		)

		if err != nil {
			return errors.Wrap(err, "fetch chart")
		}

		err = r.Templater.Template(chartLocation, rootFs, asset, meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "execute templating")
		}
		return nil
	}

}
