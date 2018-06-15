package docker

import (
	"context"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// A Renderer can execute a "docker" step in the lifecycle
type Renderer interface {
	Execute(
		asset api.DockerAsset,
		meta api.ReleaseMetadata,
		doWithProgress func(ch chan interface{}, debug log.Logger) error,
		// kind of a hack/shortcut, the abstraction is leaking,
		// but we reuse this step in dockerlayer,
		// so allow for overriding the save destination
		saveDest string,
	) func(ctx context.Context) error
}

// DefaultStep is the default implementation of Renderer
type DefaultStep struct {
	Logger      log.Logger
	Fs          afero.Afero
	URLResolver PullURLResolver
	ImageSaver  ImageSaver
	Viper       *viper.Viper
}

// NewStep gets a new Renderer with the default impl
func NewStep(
	logger log.Logger,
	fs afero.Afero,
	resolver PullURLResolver,
	saver ImageSaver,
	v *viper.Viper,
) Renderer {
	return &DefaultStep{
		Logger:      logger,
		Fs:          fs,
		URLResolver: resolver,
		ImageSaver:  saver,
		Viper:       v,
	}
}

// Execute runs the step for an asset
func (p *DefaultStep) Execute(
	asset api.DockerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, debug log.Logger) error,
	dest string,
) func(ctx context.Context) error {

	if dest == "" {
		dest = asset.Dest
	}

	return func(ctx context.Context) error {
		debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "docker", "dest", dest, "description", asset.Description))
		debug.Log("event", "execute")

		basePath := filepath.Dir(dest)
		debug.Log("event", "mkdirall.attempt", "dest", dest, "basePath", basePath)
		if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "dest", dest, "basePath", basePath)
			return errors.Wrapf(err, "write directory to %s", dest)
		}

		pullURL, err := p.URLResolver.ResolvePullURL(asset, meta)
		if err != nil {
			return errors.Wrapf(err, "resolve pull url")
		}

		// first try with registry secret
		// TODO remove this once registry is updated to read installation ID
		registrySecretSaveOpts := SaveOpts{
			PullURL:   pullURL,
			SaveURL:   asset.Image,
			IsPrivate: asset.Source != "public" && asset.Source != "",
			Filename:  dest,
			Username:  meta.CustomerID,
			Password:  meta.RegistrySecret,
		}
		ch := p.ImageSaver.SaveImage(ctx, registrySecretSaveOpts)
		saveError := doWithProgress(ch, debug)

		if saveError == nil {
			debug.Log("event", "execute.succeed")
			return nil
		}

		debug.Log("event", "execute.fail.withRegistrySecret", "err", saveError)
		debug.Log("event", "execute.try.withInstallationID")

		// next try with installationID for password
		installationIDSaveOpts := SaveOpts{
			PullURL:   pullURL,
			SaveURL:   asset.Image,
			IsPrivate: asset.Source != "public" && asset.Source != "",
			Filename:  dest,
			Username:  meta.CustomerID,
			Password:  p.Viper.GetString("installation-id"),
		}

		ch = p.ImageSaver.SaveImage(ctx, installationIDSaveOpts)
		saveError = doWithProgress(ch, debug)

		if saveError != nil {
			debug.Log("event", "execute.fail.withInstallationID", "detail", "both docker auth methods failed", "err", saveError)
			return errors.Wrap(saveError, "docker save image, both auth methods failed")
		}

		return nil
	}
}
