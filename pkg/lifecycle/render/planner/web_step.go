package planner

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"encoding/base64"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedhq/libyaml"
)

func (p *CLIPlanner) webStep(web *api.WebAsset, configGroups []libyaml.ConfigGroup, templateContext map[string]interface{}) Step {
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

			body, err := pullWebAsset(web, configCtx)
			if err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Get web asset from", web.Dest)
			}

			basePath := filepath.Dir(web.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", web.Dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", web.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", web.Dest)
			}

			mode := os.FileMode(0644)
			if web.Mode != os.FileMode(0) {
				debug.Log("event", "applying override permissions")
				mode = web.Mode
			}

			// TODO: write raw html or bytes to file?
			if err := p.Fs.WriteFile(web.Dest, body, mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write web asset to %s", web.Dest)
			}

			return nil
		},
	}
}

func pullWebAsset(web *api.WebAsset, ctx *templates.ConfigCtx) ([]byte, error) {
	client := &http.Client{}

	req, reqErr := http.NewRequest("GET", web.URL, nil)
	if reqErr != nil {
		return nil, errors.Wrapf(reqErr, "Request web asset from %s", web.URL)
	}

	fmt.Println(ctx)

	if len(web.Headers) != 0 {
		for header := range web.Headers {
			for value := range header {
				req.Header.Add(header, base64.StdEncoding.EncodeToString([]byte(string(value))))
			}
		}
	}

	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, errors.Wrapf(respErr, "Get web asset from %s", web.URL)
	}
	defer resp.Body.Close()

	bodyToBytes, byteErr := ioutil.ReadAll(resp.Body)
	if byteErr != nil {
		return nil, errors.Wrapf(respErr, "Decode response body")
	}

	return bodyToBytes, nil
}
