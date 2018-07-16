package web

import (
	"context"
	"encoding/base64"
	"net/http"

	"path/filepath"

	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Renderer interface {
	Execute(
		asset api.WebAsset,
		meta api.ReleaseMetadata,
		configGroups []libyaml.ConfigGroup,
		templateContext map[string]interface{},
	) func(ctx context.Context) error
}

var _ Renderer = &DefaultStep{}

type DefaultStep struct {
	Logger         log.Logger
	Fs             afero.Afero
	Viper          *viper.Viper
	BuilderBuilder *templates.BuilderBuilder
	Client         *http.Client
}

func NewStep(
	logger log.Logger,
	fs afero.Afero,
	v *viper.Viper,
	builderBuilder *templates.BuilderBuilder,
) Renderer {
	return &DefaultStep{
		Logger:         logger,
		Fs:             fs,
		Viper:          v,
		BuilderBuilder: builderBuilder,
		Client:         &http.Client{},
	}
}

type Built struct {
	URL     string
	Dest    string
	Headers map[string][]string
}

func (p *DefaultStep) Execute(
	asset api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) func(ctx context.Context) error {

	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute",
		"asset.type", "web", "dest", asset.Dest, "description", asset.Description))

	return func(ctx context.Context) error {
		debug.Log("event", "execute")

		built, err := p.buildAsset(asset, meta, configGroups, templateContext)
		if err != nil {
			debug.Log("event", "build.fail", "err", err)
			return errors.Wrapf(err, "Build web asset")
		}

		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "dest", built.Dest, "basePath", basePath)
		if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "dest", built.Dest, "basePath", basePath)
			return errors.Wrapf(err, "Create directory path %s", basePath)
		}

		if err := p.pullWebAsset(built); err != nil {
			debug.Log("event", "pullWebAsset.fail", "err", err)
			return errors.Wrapf(err, "Get web asset from %s", asset.URL)
		}

		return nil
	}
}

func (p *DefaultStep) buildAsset(
	asset api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) (*Built, error) {
	configCtx, err := p.BuilderBuilder.NewConfigContext(configGroups, templateContext)
	if err != nil {
		return nil, errors.Wrap(err, "getting config context")
	}

	builder := p.BuilderBuilder.NewBuilder(
		p.BuilderBuilder.NewStaticContext(),
		configCtx,
		&templates.InstallationContext{
			Meta:  meta,
			Viper: p.Viper,
		},
	)

	builtURL, err := builder.String(asset.URL)
	if err != nil {
		return nil, errors.Wrap(err, "building url")
	}

	builtDest, err := builder.String(asset.Dest)
	if err != nil {
		return nil, errors.Wrap(err, "building dest")
	}

	builtHeaders := make(map[string][]string)
	for header, listOfValues := range asset.Headers {
		for _, value := range listOfValues {
			builtHeaderVal, err := builder.String(value)
			if err != nil {
				return nil, errors.Wrap(err, "building header val")
			}
			builtHeaders[header] = append(builtHeaders[header], builtHeaderVal)
		}
	}
	return &Built{
		URL:     builtURL,
		Dest:    builtDest,
		Headers: builtHeaders,
	}, nil
}

func (p *DefaultStep) pullWebAsset(built *Built) error {
	req, err := http.NewRequest("GET", built.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "Request web asset from %s", built.URL)
	}

	if len(built.Headers) != 0 {
		for header, listOfValues := range built.Headers {
			for _, value := range listOfValues {
				req.Header.Add(header, base64.StdEncoding.EncodeToString([]byte(value)))
			}
		}
	}

	resp, respErr := p.Client.Do(req)
	if respErr != nil {
		return errors.Wrapf(respErr, "%s web asset at %s", "GET", built.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return errors.Errorf("received response with status %d", resp.StatusCode)
	}

	file, err := p.Fs.Create(built.Dest)
	if err != nil {
		return errors.Wrapf(err, "Create file %s", file)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return errors.Wrapf(err, "Stream HTTP response body to %s", file.Name())
	}
	file.Close()

	return nil
}
