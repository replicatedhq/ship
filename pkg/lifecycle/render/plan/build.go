package plan

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"text/template"

	"os"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
)

// Build builds a plan in memory from assets+resolved config
func (p *CLIPlanner) Build(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) Plan {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "phase", "plan"))
	var plan Plan
	for _, asset := range assets {
		if asset.Inline != nil {
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, p.inlineStep(asset.Inline, meta, templateContext))
		} else if asset.Docker != nil {
			debug.Log("event", "asset.resolve", "asset.type", "docker")
			plan = append(plan, p.dockerStep(asset.Docker, meta, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
		}
	}
	return plan
}

func (p *CLIPlanner) inlineStep(inline *api.InlineAsset, _ api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", inline.Dest, "description", inline.Description))
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")
			tpl, err := template.New(inline.Description).
				Delims("{{repl ", "}}").
				Funcs(p.funcMap(templateContext)).
				Parse(inline.Contents)
			if err != nil {
				return errors.Wrapf(err, "Parse template for asset at %s", inline.Dest)
			}

			var rendered bytes.Buffer
			err = tpl.Execute(&rendered, templateContext)
			if err != nil {
				return errors.Wrapf(err, "Execute template for asset at %s", inline.Dest)
			}

			basePath := filepath.Dir(inline.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", inline.Dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", inline.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", inline.Dest)
			}

			mode := os.FileMode(644)
			if inline.Mode != 0 {
				mode = inline.Mode
			}

			if err := p.Fs.WriteFile(inline.Dest, rendered.Bytes(), mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write inline asset to %s", inline.Dest)
			}
			return nil
		},
	}
}

func (p *CLIPlanner) dockerStep(asset *api.DockerAsset, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "docker", "dest", asset.Dest, "description", asset.Description))
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")
			basePath := filepath.Dir(asset.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", asset.Dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", asset.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", asset.Dest)
			}

			authOpts := dockertypes.AuthConfig{}
			if asset.Source == "replicated" {
				authOpts.Username = meta.CustomerID
				authOpts.Password = meta.RegistrySecret
			}

			if err := docker.SaveImage(ctx, p.Logger, asset.Image, asset.Dest, authOpts); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write docker asset to %s", asset.Dest)
			}

			return nil
		},
	}
}

func (p *CLIPlanner) funcMap(templateContext map[string]interface{}) template.FuncMap {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "template"))

	configFunc := func(name string) interface{} {
		configItemValue, ok := templateContext[name]
		if !ok {
			debug.Log("event", "template.missing", "func", "config", "requested", name, "context", templateContext)
			return ""
		}
		return configItemValue
	}

	return map[string]interface{}{
		"config":       configFunc,
		"ConfigOption": configFunc,
		"context": func(name string) interface{} {
			switch name {
			case "state_file_path":
				return state.Path
			}
			debug.Log("event", "template.missing", "func", "context", "requested", name)
			return ""
		},
	}
}
