package docker

import (
	"context"
	"net/url"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/images"
	"github.com/replicatedhq/ship/pkg/templates"
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
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

var _ Renderer = &DefaultStep{}

// DefaultStep is the default implementation of Renderer
type DefaultStep struct {
	Logger         log.Logger
	Fs             afero.Afero
	URLResolver    images.PullURLResolver
	ImageSaver     images.ImageSaver
	Viper          *viper.Viper
	BuilderBuilder *templates.BuilderBuilder
}

// NewStep gets a new Renderer with the default impl
func NewStep(
	logger log.Logger,
	fs afero.Afero,
	resolver images.PullURLResolver,
	saver images.ImageSaver,
	v *viper.Viper,
	bb *templates.BuilderBuilder,
) Renderer {
	return &DefaultStep{
		Logger:         logger,
		Fs:             fs,
		URLResolver:    resolver,
		ImageSaver:     saver,
		Viper:          v,
		BuilderBuilder: bb,
	}
}

// Execute runs the step for an asset
func (p *DefaultStep) Execute(
	asset api.DockerAsset,
	meta api.ReleaseMetadata,
	doWithProgress func(ch chan interface{}, debug log.Logger) error,
	dest string,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {

	if dest == "" {
		dest = asset.Dest
	}

	return func(ctx context.Context) error {
		debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "docker", "dest", dest, "description", asset.Description))
		debug.Log("event", "execute")
		configCtx, err := p.BuilderBuilder.NewConfigContext(configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "create config context")
		}

		builder := p.BuilderBuilder.NewBuilder(
			p.BuilderBuilder.NewStaticContext(),
			configCtx,
			&templates.InstallationContext{
				Meta:  meta,
				Viper: p.Viper,
			},
		)
		builtDest, err := builder.String(dest)
		if err != nil {
			return errors.Wrap(err, "building dest")
		}

		destinationURL, err := url.Parse(builtDest)
		if err != nil {
			return errors.Wrapf(err, "parse destination URL %s", dest)
		}
		destIsDockerURL := destinationURL.Scheme == "docker"
		if !destIsDockerURL {
			dest = filepath.Join(constants.InstallerPrefix, dest)
			basePath := filepath.Dir(dest)
			debug.Log("event", "mkdirall.attempt", "dest", dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", dest)
			}
		}

		pullURL, err := p.URLResolver.ResolvePullURL(asset, meta)
		if err != nil {
			return errors.Wrapf(err, "resolve pull url")
		}

		// first try with registry secret
		// TODO remove this once registry is updated to read installation ID
		registrySecretSaveOpts := images.SaveOpts{
			PullURL:   pullURL,
			SaveURL:   asset.Image,
			IsPrivate: asset.Source != "public" && asset.Source != "",
			Username:  meta.CustomerID,
			Password:  meta.RegistrySecret,
		}

		if destIsDockerURL {
			registrySecretSaveOpts.DestinationURL = destinationURL
		} else {
			registrySecretSaveOpts.Filename = dest
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
		installationIDSaveOpts := images.SaveOpts{
			PullURL:   pullURL,
			SaveURL:   asset.Image,
			IsPrivate: asset.Source != "public" && asset.Source != "",
			Username:  meta.CustomerID,
			Password:  p.Viper.GetString("installation-id"),
		}

		if destIsDockerURL {
			installationIDSaveOpts.DestinationURL = destinationURL
		} else {
			installationIDSaveOpts.Filename = dest
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
