package dockerlayer

import (
	"context"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/util"
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
	rootFs root.Fs,
	asset api.DockerLayerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, logger log.Logger) error,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(context.Context) error {
	return func(ctx context.Context) error {
		debug := level.Debug(log.With(u.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "dockerlayer", "dest", asset.Dest, "description", asset.Description))

		if err := u.mkdirall(rootFs, "tmp")(); err != nil {
			return errors.Wrap(err, "create root tmp path dir")
		}
		savePath, firstPassUnpackPath, basePath, layerPath, err := u.getPaths(asset, rootFs)
		defer rootFs.RemoveAll("tmp") // nolint: errcheck
		if err != nil {
			return errors.Wrap(err, "resolve unpack paths")
		}

		err = util.IsLegalPath(basePath)
		if err != nil {
			return errors.Wrap(err, "write docker layer")
		}

		debug.Log(
			"event", "execute",
			"savePath", savePath,
			"firstUnpack", firstPassUnpackPath,
			"basePath", basePath,
			"layerPath", layerPath,
		)

		return errors.Wrap(u.chain(
			u.save(ctx, rootFs, asset, meta, doWithProgress, savePath, templateContext, configGroups),
			u.mkdirall(rootFs, basePath),
			u.unpack(rootFs, savePath, firstPassUnpackPath),
			u.unpack(rootFs, layerPath, basePath),
		), "execute chain")
	}
}

func (u *Unpacker) getPaths(asset api.DockerLayerAsset, rootFs root.Fs) (string, string, string, string, error) {
	fail := func(err error) (string, string, string, string, error) { return "", "", "", "", err }

	saveDir, err := rootFs.TempDir("/tmp", "dockerlayer")
	if err != nil {
		return fail(errors.Wrap(err, "get image save tmpdir"))
	}

	savePath := path.Join(saveDir, "image.tar")

	firstPassUnpackPath, err := rootFs.TempDir("/tmp", "dockerlayer")
	if err != nil {
		return fail(errors.Wrap(err, "get unpack tmpdir"))
	}

	basePath := asset.Dest //TODO enforce that this is a directory
	layerPath := path.Join(firstPassUnpackPath, asset.Layer, "layer.tar")
	return savePath, firstPassUnpackPath, basePath, layerPath, nil
}

func (u *Unpacker) save(
	ctx context.Context,
	rootFs root.Fs,
	asset api.DockerLayerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, logger log.Logger) error,
	savePath string,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func() error {
	return func() error {
		return errors.Wrapf(
			u.DockerSaver.Execute(
				rootFs,
				asset.DockerAsset,
				meta,
				doWithProgress,
				savePath,
				templateContext,
				configGroups,
			)(ctx),
			"save image to %s ", savePath)
	}
}

func (u *Unpacker) unpack(rootFs root.Fs, src string, dest string) func() error {
	return func() error {
		rootPathedSrc := path.Join(rootFs.RootPath, src)
		rootPathedDest := path.Join(rootFs.RootPath, dest)
		return errors.Wrapf(u.Tar.Open(rootPathedSrc, rootPathedDest), "untar %s to %s", rootPathedSrc, rootPathedDest)
	}
}

func (u *Unpacker) mkdirall(rootFs root.Fs, basePath string) func() error {
	return func() error {
		return errors.Wrapf(rootFs.MkdirAll(basePath, 0755), "mkdirall %s", basePath)
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
