package planner

import (
	"context"
	"fmt"
	"path/filepath"

	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedhq/libyaml"
)

// Build builds a plan in memory from assets+resolved config
func (p *CLIPlanner) Build(assets []api.Asset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Plan {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "phase", "plan"))
	var plan Plan
	for _, asset := range assets {
		if asset.Inline != nil {
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, p.inlineStep(asset.Inline, configGroups, meta, templateContext))
		} else if asset.Docker != nil {
			debug.Log("event", "asset.resolve", "asset.type", "docker")
			plan = append(plan, p.dockerStep(asset.Docker, meta, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
		}
	}
	return plan
}

func (p *CLIPlanner) inlineStep(inline *api.InlineAsset, configGroups []libyaml.ConfigGroup, r api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", inline.Dest, "description", inline.Description))
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")

			staticCtx, err := config.NewStaticContext()
			if err != nil {
				return errors.Wrap(err, "getting static context")
			}

			configCtx, err := config.NewConfigContext(
				p.Viper, p.Logger,
				configGroups, templateContext)
			if err != nil {
				return errors.Wrap(err, "getting config context")
			}

			builder := config.NewBuilder(
				staticCtx,
				configCtx,
			)

			built, err := builder.String(inline.Contents)
			if err != nil {
				return errors.Wrap(err, "building contents")
			}

			basePath := filepath.Dir(inline.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", inline.Dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", inline.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", inline.Dest)
			}

			mode := os.FileMode(0644)
			if inline.Mode != os.FileMode(0) {
				debug.Log("event", "applying override permissions")
				mode = inline.Mode
			}

			if err := p.Fs.WriteFile(inline.Dest, []byte(built), mode); err != nil {
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

			pullUrl, err := docker.ResolvePullUrl(asset, meta)
			if err != nil {
				return errors.Wrapf(err, "resolve pull url")
			}

			saveOpts := docker.SaveOpts{
				PullUrl:   pullUrl,
				SaveUrl:   asset.Image,
				IsPrivate: asset.Source != "public" && asset.Source != "",
				Filename:  asset.Dest,
				Username:  meta.CustomerID,
				Password:  meta.RegistrySecret,
			}

			if err := docker.SaveImage(ctx, p.Logger, saveOpts); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write docker asset to %s", asset.Dest)
			}

			return nil
		},
	}
}
