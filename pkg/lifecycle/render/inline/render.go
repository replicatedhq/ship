package inline

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/viper"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.InlineAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

var _ Renderer = &LocalRenderer{}

// LocalRenderer can add a helm step to the plan, the step will fetch the
// chart to a temporary location and then run a local operation to run the helm templating
type LocalRenderer struct {
	Logger         log.Logger
	BuilderBuilder *templates.BuilderBuilder
	Viper          *viper.Viper
}

func NewRenderer(
	logger log.Logger,
	bb *templates.BuilderBuilder,
	v *viper.Viper,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		BuilderBuilder: bb,
		Viper:          v,
	}
}

func (r *LocalRenderer) Execute(
	rootFs root.Fs,
	asset api.InlineAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", asset.Dest, "description", asset.Description))

	return func(ctx context.Context) error {
		debug.Log("event", "execute")

		builder, err := r.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "init builder")
		}

		builtAsset, err := templateInline(builder, asset)
		if err != nil {
			return errors.Wrap(err, "building contents")
		}

		err = util.IsLegalPath(builtAsset.Dest)
		if err != nil {
			return errors.Wrap(err, "write inline asset")
		}

		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "dest", builtAsset.Dest, "basePath", basePath)
		if err := rootFs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "dest", builtAsset.Dest, "basePath", basePath)
			return errors.Wrapf(err, "write directory to %s", builtAsset.Dest)
		}

		mode := os.FileMode(0644)
		if builtAsset.Mode != os.FileMode(0) {
			debug.Log("event", "applying override permissions")
			mode = builtAsset.Mode
		}

		if err := rootFs.WriteFile(builtAsset.Dest, []byte(builtAsset.Contents), mode); err != nil {
			debug.Log("event", "execute.fail", "err", err)
			return errors.Wrapf(err, "Write inline asset to %s", builtAsset.Dest)
		}
		return nil

	}
}

func templateInline(builder *templates.Builder, asset api.InlineAsset) (api.InlineAsset, error) {
	builtAsset := asset
	var err error

	builtAsset.Contents, err = builder.String(asset.Contents)
	if err != nil {
		return builtAsset, errors.Wrap(err, "building contents")
	}

	builtAsset.Dest, err = builder.String(asset.Dest)
	if err != nil {
		return builtAsset, errors.Wrap(err, "building dest")
	}

	return builtAsset, nil
}
