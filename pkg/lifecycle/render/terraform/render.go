package terraform

import (
	"context"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.TerraformAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// a LocalRenderer renders a terraform asset by vendoring in terraform source code
type LocalRenderer struct {
	Logger log.Logger
	Inline inline.Renderer
	Fs     afero.Afero
}

var _ Renderer = &LocalRenderer{}

func NewRenderer(
	logger log.Logger,
	inline inline.Renderer,
	fs afero.Afero,
) Renderer {
	return &LocalRenderer{
		Logger: logger,
		Inline: inline,
		Fs:     fs,
	}
}

func (r *LocalRenderer) Execute(
	asset api.TerraformAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		if asset.Inline == "" {
			return errors.New("online \"inline\" terraform assets are supported")
		}

		assetsPath := path.Join("terraform", "main.tf")

		// write the inline spec
		err := r.Inline.Execute(
			api.InlineAsset{
				Contents: asset.Inline,
				AssetShared: api.AssetShared{
					Dest: assetsPath,
				},
			},
			meta,
			templateContext,
			configGroups,
		)(ctx)

		if err != nil {
			return errors.Wrap(err, "write tf config")
		}
		return nil
	}
}
