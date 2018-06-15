package dockerlayer

import (
	"context"

	"path"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// An unpacker
type Unpacker struct {
	Logger      log.Logger
	FS          afero.Afero
	Viper       *viper.Viper
	DockerSaver docker.Renderer
	Tar         archiver.Archiver
}

func TarArchiver() archiver.Archiver {
	return archiver.Tar
}

func NewUnpacker(
	logger log.Logger,
	dockerStep docker.Renderer,
	fs afero.Afero,
	viper *viper.Viper,
	tar archiver.Archiver,
) *Unpacker {

	return &Unpacker{
		Logger:      logger,
		FS:          fs,
		Viper:       viper,
		DockerSaver: dockerStep,
		Tar:         tar,
	}
}

func (u *Unpacker) Execute(
	asset api.DockerLayerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, logger log.Logger) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		debug := level.Debug(log.With(u.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "dockerlayer", "dest", asset.Dest, "description", asset.Description))

		savePath, firstPassUnpackPath, basePath, layerPath, err := u.getPaths(asset)
		if err != nil {
			return errors.Wrap(err, "resolve unpack paths")
		}

		debug.Log(
			"event", "execute",
			"savePath", savePath,
			"firstUnpack", firstPassUnpackPath,
			"basePath", basePath,
			"layerPath", layerPath,
		)

		return errors.Wrap(u.chain(
			u.save(ctx, asset, meta, doWithProgress, savePath),
			u.mkdirall(basePath),
			u.unpack(savePath, firstPassUnpackPath),
			u.unpack(layerPath, asset.Dest),
		), "execute chain")

	}
}

func (u *Unpacker) getPaths(asset api.DockerLayerAsset) (string, string, string, string, error) {
	fail := func(err error) (string, string, string, string, error) { return "", "", "", "", err }
	saveDir, err := u.FS.TempDir("/tmp", "dockerlayer")
	if err != nil {
		return fail(err)
	}

	savePath := path.Join(saveDir, "image.tar")

	firstPassUnpackPath, err := u.FS.TempDir("/tmp", "dockerlayer")
	if err != nil {
		return fail(err)
	}

	basePath := filepath.Dir(asset.Dest)
	layerPath := path.Join(firstPassUnpackPath, asset.Layer, "layer.tar")
	return savePath, firstPassUnpackPath, basePath, layerPath, nil
}

func (u *Unpacker) save(
	ctx context.Context,
	asset api.DockerLayerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, logger log.Logger) error,
	savePath string,
) func() error {
	return func() error {
		return errors.Wrapf(
			u.DockerSaver.Execute(
				asset.DockerAsset,
				meta,
				doWithProgress,
				savePath,
			)(ctx),
			"save image to %s ", savePath)
	}
}

func (u *Unpacker) unpack(src string, dest string) func() error {
	return func() error {
		return errors.Wrapf(u.Tar.Open(src, dest), "untar %s to %s", src, dest)
	}
}

func (u *Unpacker) mkdirall(basePath string) func() error {
	return func() error {
		return errors.Wrapf(u.FS.MkdirAll(basePath, 0755), "mkdirall %s", basePath)
	}
}

// this is here because it makes sense here, not trying to reinvent any wheels
func (u *Unpacker) chain(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}
