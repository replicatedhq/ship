package planner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedcom/ship/pkg/templates"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type buildProgress struct {
	StepNumber int `json:"step_number"`
	TotalSteps int `json:"total_steps"`
}

// Build builds a plan in memory from assets+resolved config
func (p *CLIPlanner) Build(assets []api.Asset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Plan {
	defer p.Daemon.ClearProgress()

	debug := level.Debug(log.With(p.Logger, "step.type", "render", "phase", "plan"))
	var plan Plan
	for i, asset := range assets {
		progress := buildProgress{
			StepNumber: i,
			TotalSteps: len(assets),
		}
		p.Daemon.SetProgress(config.JSONProgress("build", progress))

		if asset.Inline != nil {
			asset.Inline.Dest = filepath.Join("installer", asset.Inline.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, p.inlineStep(asset.Inline, configGroups, meta, templateContext))
		} else if asset.Docker != nil {
			asset.Docker.Dest = filepath.Join("installer", asset.Docker.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "docker")
			plan = append(plan, p.dockerStep(asset.Docker, meta, templateContext))
		} else if asset.Web != nil {
			asset.Web.Dest = filepath.Join("installer", asset.Web.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "web")
			plan = append(plan, p.webStep(asset.Web, configGroups, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
		}
	}
	return plan
}

func (p *CLIPlanner) inlineStep(inline *api.InlineAsset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", inline.Dest, "description", inline.Description))
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")

			configCtx, err := templates.NewConfigContext(
				p.Viper, p.Logger,
				configGroups, templateContext)
			if err != nil {
				return errors.Wrap(err, "getting config context")
			}

			fmt.Println("*********")
			fmt.Println(configCtx)
			fmt.Println("*********")

			builder := p.BuilderBuilder.NewBuilder(
				templates.NewStaticContext(),
				configCtx,
				&templates.InstallationContext{
					Meta:  meta,
					Viper: p.Viper,
				},
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
				Logger:    p.Logger,
			}

			ch := docker.SaveImage(ctx, saveOpts)

			var saveError error
			for msg := range ch {
				if msg == nil {
					continue
				}
				switch v := msg.(type) {
				case error:
					// continue reading on error to ensure channel is not blocked
					saveError = v
				case docker.DockerProgress:
					p.Daemon.SetProgress(config.JSONProgress("docker", v))
				case string:
					p.Daemon.SetProgress(config.StringProgress("docker", v))
				default:
					debug.Log("event", "progress", "message", fmt.Sprintf("%#v", v))
				}
			}

			if saveError != nil {
				debug.Log("event", "execute.fail", "err", saveError)
				return errors.Wrapf(saveError, "Write docker asset to %s", asset.Dest)
			}

			return nil
		},
	}
}
