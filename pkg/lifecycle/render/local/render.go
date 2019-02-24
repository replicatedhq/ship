package local

import (
	"context"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"

	"github.com/spf13/afero"
)

type Renderer interface {
	Execute(
		asset api.LocalAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

var _ Renderer = &LocalRenderer{}

type LocalRenderer struct {
	Logger log.Logger
	Fs     afero.Afero
}

func NewRenderer(
	logger log.Logger,
	fs afero.Afero,
) Renderer {
	return &LocalRenderer{
		Logger: logger,
		Fs:     fs,
	}
}

func (r *LocalRenderer) Execute(
	asset api.LocalAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "local"))

		err := util.IsLegalPath(asset.Dest)
		if err != nil {
			return errors.Wrap(err, "local asset dest")
		}

		err = util.IsLegalPath(asset.Path)
		if err != nil {
			return errors.Wrap(err, "local asset path")
		}

		if err := r.Fs.MkdirAll(filepath.Dir(asset.Dest), 0777); err != nil {
			return errors.Wrapf(err, "mkdir %s", asset.Dest)
		}

		debug.Log("event", "rename", "from", asset.Path, "dest", asset.Dest)
		if err := r.Fs.Rename(asset.Path, asset.Dest); err != nil {
			return errors.Wrap(err, "rename path to dest")
		}

		return nil
	}
}
