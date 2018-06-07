package planner

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"encoding/base64"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedhq/libyaml"
)

func (p *CLIPlanner) webStep(web *api.WebAsset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "web", "dest", web.Dest, "description", web.Description))
	return Step{
		Dest:        web.Dest,
		Description: web.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")

			configCtx, err := templates.NewConfigContext(
				p.Viper, p.Logger,
				configGroups, templateContext)
			if err != nil {
				return errors.Wrap(err, "getting config context")
			}

			builder := p.BuilderBuilder.NewBuilder(
				templates.NewStaticContext(),
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

			builtHeaders := make(map[string][]string)
			for header, listOfValues := range web.Headers {
				for _, value := range listOfValues {
					builtHeaderVal, err := builder.String(value)
					if err != nil {
						return errors.Wrap(err, "building header val")
					}
					builtHeaders[header] = append(web.Headers[header], builtHeaderVal)
				}
			}

			body, err := pullWebAsset(builtUrl, builtHeaders)
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

func pullWebAsset(url string, headers map[string][]string) ([]byte, error) {
	client := &http.Client{}

	req, reqErr := http.NewRequest("GET", url, nil)
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

	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, errors.Wrapf(respErr, "Get web asset from %s", url)
	}
	defer resp.Body.Close()

	bodyToBytes, byteErr := ioutil.ReadAll(resp.Body)
	if byteErr != nil {
		return nil, errors.Wrapf(respErr, "Decode response body")
	}

	return bodyToBytes, nil
}
