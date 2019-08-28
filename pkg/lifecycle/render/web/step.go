package web

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.WebAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
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
	URL        string
	Dest       string
	Method     string
	Body       string
	Headers    map[string][]string
	BodyFormat string
}

func (p *DefaultStep) Execute(
	rootFs root.Fs,
	asset api.WebAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {

	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute",
		"asset.type", "web", "dest", asset.Dest, "description", asset.Description))

	return func(ctx context.Context) error {
		debug.Log("event", "execute")

		if p.Viper.GetBool("no-web") {
			return errors.New("web assets are disabled when no-web is set")
		}

		built, err := p.buildAsset(asset, meta, configGroups, templateContext)
		if err != nil {
			debug.Log("event", "build.fail", "err", err)
			return errors.Wrapf(err, "Build web asset")
		}

		err = util.IsLegalPath(built.Dest)
		if err != nil {
			return errors.Wrap(err, "write web asset")
		}

		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "root", rootFs.RootPath, "dest", built.Dest, "basePath", basePath)
		if err := rootFs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "root", rootFs.RootPath, "dest", built.Dest, "basePath", basePath)
			return errors.Wrapf(err, "Create directory path %s", basePath)
		}

		if err := p.pullWebAsset(rootFs, built); err != nil {
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

	builder, err := p.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
	if err != nil {
		return nil, errors.Wrap(err, "init builder")
	}

	builtURL, err := builder.String(asset.URL)
	if err != nil {
		return nil, errors.Wrap(err, "building url")
	}

	builtDest, err := builder.String(asset.Dest)
	if err != nil {
		return nil, errors.Wrap(err, "building dest")
	}

	builtMethod, err := builder.String(asset.Method)
	if err != nil {
		return nil, errors.Wrap(err, "building method")
	}

	builtBody, err := builder.String(asset.Body)
	if err != nil {
		return nil, errors.Wrap(err, "building body")
	}

	builtBodyFormat, err := builder.String(asset.BodyFormat)
	if err != nil {
		return nil, errors.Wrap(err, "building content type")
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
		URL:        builtURL,
		Dest:       builtDest,
		Method:     builtMethod,
		Body:       builtBody,
		BodyFormat: builtBodyFormat,
		Headers:    builtHeaders,
	}, nil
}

func (p *DefaultStep) pullWebAsset(rootFs root.Fs, built *Built) error {
	req, err := http.NewRequest(built.Method, built.URL, strings.NewReader(built.Body))
	if err != nil {
		return errors.Wrapf(err, "Request web asset from %s", built.URL)
	}

	if built.Method == "POST" {
		req.Header.Set("Content-Type", built.BodyFormat)
	}

	if len(built.Headers) != 0 {
		for header, listOfValues := range built.Headers {
			for _, value := range listOfValues {
				req.Header.Add(header, value)
			}
		}
	}

	resp, respErr := p.Client.Do(req)
	if respErr != nil {
		return errors.Wrapf(respErr, "%s web asset at %s", built.Method, built.URL)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode > 299 {
		return errors.Errorf("received response with status %d", resp.StatusCode)
	}

	file, err := rootFs.Create(built.Dest)
	if err != nil {
		return errors.Wrapf(err, "Create file %s", file)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return errors.Wrapf(err, "Stream HTTP response body to %s", file.Name())
	}
	return errors.Wrapf(file.Close(), "close file at %s", built.Dest)
}
