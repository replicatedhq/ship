package planner

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"encoding/base64"
	"encoding/json"

	"bytes"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
)

func (p *CLIPlanner) webStep(web *api.WebAsset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "web", "dest", web.Dest, "description", web.Description))
	return Step{
		Dest:        web.Dest,
		Description: web.Description,
		Execute: func(ctx context.Context) error {
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

			builtUrl, err := builder.String(web.URL)
			if err != nil {
				return errors.Wrap(err, "building url")
			}

			builtDest, err := builder.String(web.Dest)
			if err != nil {
				return errors.Wrap(err, "building dest")
			}

			builtMethod, err := builder.String(web.Method)
			if err != nil {
				return errors.Wrap(err, "building method")
			}

			builtBody, err := builder.String(web.Body)
			if err != nil {
				return errors.Wrap(err, "building body")
			}

			builtHeaders := make(map[string][]string)
			for header, listOfValues := range web.Headers {
				for _, value := range listOfValues {
					builtHeaderVal, err := builder.String(value)
					if err != nil {
						return errors.Wrap(err, "building header val")
					}
					builtHeaders[header] = append(builtHeaders[header], builtHeaderVal)
				}
			}

			body, err := pullWebAsset(builtUrl, builtMethod, builtBody, builtHeaders)
			if err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Get web asset from", web.Dest)
			}

			basePath := filepath.Dir(web.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", builtDest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", builtDest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", builtDest)
			}

			mode := os.FileMode(0644)
			if web.Mode != os.FileMode(0) {
				debug.Log("event", "applying override permissions")
				mode = web.Mode
			}

			if err := p.Fs.WriteFile(builtDest, body, mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write web asset to %s", builtDest)
			}

			return nil
		},
	}
}

func pullWebAsset(url string, method string, body string, headers map[string][]string) ([]byte, error) {
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

	bodyToBytes, byteErr := ioutil.ReadAll(resp.Body)
	if byteErr != nil {
		return nil, errors.Wrapf(respErr, "Decode response body")
	}

	return bodyToBytes, nil
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
