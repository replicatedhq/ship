package web

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"encoding/base64"
	"encoding/json"

	"bytes"

	"io/ioutil"

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

type DefaultStep struct {
	Logger         log.Logger
	Fs             afero.Afero
	Viper          *viper.Viper
	BuilderBuilder *templates.BuilderBuilder
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
	}
}

func (p *DefaultStep) Execute(
	asset api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) func(ctx context.Context) error {

	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "web", "dest", web.Dest, "description", web.Description))
	return func(ctx context.Context) error {
		debug.Log("event", "execute")

		configCtx, err := templates.NewConfigContext(p.Logger, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "getting config context")
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
			return errors.Wrap(err, "building url")
		}

		builtDest, err := builder.String(asset.Dest)
		if err != nil {
			return errors.Wrap(err, "building dest")
		}

		builtMethod, err := builder.String(asset.Method)
		if err != nil {
			return errors.Wrap(err, "building method")
		}

		builtBody, err := builder.String(asset.Body)
		if err != nil {
			return errors.Wrap(err, "building body")
		}

		builtHeaders := make(map[string][]string)
		for header, listOfValues := range asset.Headers {
			for _, value := range listOfValues {
				builtHeaderVal, err := builder.String(value)
				if err != nil {
					return errors.Wrap(err, "building header val")
				}
				builtHeaders[header] = append(builtHeaders[header], builtHeaderVal)
			}
		}

		body, err := pullWebAsset(builtURL, builtMethod, builtBody, builtHeaders)
		if err != nil {
			debug.Log("event", "execute.fail", "err", err)
			return errors.Wrapf(err, "Get web asset from", asset.Dest)
		}

		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "dest", builtDest, "basePath", basePath)
		if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "dest", builtDest, "basePath", basePath)
			return errors.Wrapf(err, "write directory to %s", builtDest)
		}

		mode := os.FileMode(0644)
		if asset.Mode != os.FileMode(0) {
			debug.Log("event", "applying override permissions")
			mode = asset.Mode
		}

		bodyToBytes, byteErr := ioutil.ReadAll(body.Body)
		if byteErr != nil {
			return errors.Wrapf(byteErr, "Decode response body")
		}
		if err := p.Fs.WriteFile(builtDest, bodyToBytes, mode); err != nil {
			debug.Log("event", "execute.fail", "err", err)
			return errors.Wrapf(err, "Write web asset to %s", builtDest)
		}

		return nil
	}
}

func pullWebAsset(url string, method string, body string, headers map[string][]string) (*http.Response, error) {
	req, reqErr := parseRequest(url, method, body)
	if reqErr != nil {
		return nil, errors.Wrapf(reqErr, "Request web asset from %s", url)
	}

	if len(headers) != 0 {
		for header, listOfValues := range headers {
			for _, value := range listOfValues {
				req.Header.Add(header, base64.StdEncoding.EncodeToString([]byte(string(value))))
			}
		}
	}

	client := &http.Client{}
	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, errors.Wrapf(respErr, "%s web asset at %s", method, url)
	}
	defer resp.Body.Close()

	return resp, nil
}

func parseRequest(url string, method string, body string) (*http.Request, error) {
	switch method {
	case "GET":
		req, err := http.NewRequest("GET", url, nil)
		return req, err
	case "POST":
		jsonValue, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrapf(err, "marshal body", body)
		}
		req, err := http.NewRequest("POST", url, bytes.NewReader(jsonValue))
		return req, nil
	}
	return nil, errors.New("Parse web request")
}
