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
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
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
	FS             afero.Afero
	Viper          *viper.Viper
}

func NewRenderer(
	logger log.Logger,
	bb *templates.BuilderBuilder,
	fs afero.Afero,
	v *viper.Viper,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		BuilderBuilder: bb,
		FS:             fs,
		Viper:          v,
	}
}

func (r *LocalRenderer) Execute(
	asset api.InlineAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", asset.Dest, "description", asset.Description))

	return func(ctx context.Context) error {
		debug.Log("event", "execute")

		configCtx, err := templates.NewConfigContext(r.Logger, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "getting config context")
		}

		builder := r.BuilderBuilder.NewBuilder(
			r.BuilderBuilder.NewStaticContext(),
			configCtx,
			&templates.InstallationContext{
				Meta:  meta,
				Viper: r.Viper,
			},
		)

		built, err := builder.String(asset.Contents)
		if err != nil {
			return errors.Wrap(err, "building contents")
		}

		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "dest", asset.Dest, "basePath", basePath)
		if err := r.FS.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "dest", asset.Dest, "basePath", basePath)
			return errors.Wrapf(err, "write directory to %s", asset.Dest)
		}

		mode := os.FileMode(0644)
		if asset.Mode != os.FileMode(0) {
			debug.Log("event", "applying override permissions")
			mode = asset.Mode
		}

		if err := r.FS.WriteFile(asset.Dest, []byte(built), mode); err != nil {
			debug.Log("event", "execute.fail", "err", err)
			return errors.Wrapf(err, "Write inline asset to %s", asset.Dest)
		}
		return nil

	}
}
