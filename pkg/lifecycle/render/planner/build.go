package planner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/templates"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/images"
)

type buildProgress struct {
	StepNumber int `json:"step_number"`
	TotalSteps int `json:"total_steps"`
}

// Build builds a plan in memory from assets+resolved config
func (p *CLIPlanner) Build(assets []api.Asset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) (Plan, error) {

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
			asset.Inline.Dest = filepath.Join(constants.InstallerPrefix, asset.Inline.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, p.inlineStep(asset.Inline, configGroups, meta, templateContext))
		} else if asset.Docker != nil {
			asset.Docker.Dest = filepath.Join(constants.InstallerPrefix, asset.Docker.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "docker")
			plan = append(plan, p.dockerStep(*asset.Docker, meta))
		} else if asset.Helm != nil {
			asset.Helm.Dest = filepath.Join(constants.InstallerPrefix, asset.Helm.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "helm")
			plan = append(plan, p.helmStep(*asset.Helm, meta, templateContext, configGroups))
		} else if asset.DockerLayer != nil {
			asset.DockerLayer.Dest = filepath.Join(constants.InstallerPrefix, asset.DockerLayer.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "dockerlayer")
			plan = append(plan, p.dockerLayerStep(*asset.DockerLayer, meta))
		} else if asset.Web != nil {
			asset.Web.Dest = filepath.Join("installer", asset.Web.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "web")
			plan = append(plan, p.webStep(*asset.Web, meta, configGroups, templateContext))
		} else if asset.GitHub != nil {
			asset.GitHub.Dest = filepath.Join("installer", asset.GitHub.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "github")
			plan = append(plan, p.githubStep(*asset.GitHub, configGroups, meta, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
			return nil, errors.New("Unknown asset: type is not one of [inline docker helm dockerlayer]")
		}
	}
	return plan, nil
}

func (p *CLIPlanner) inlineStep(inline *api.InlineAsset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline", "dest", inline.Dest, "description", inline.Description))
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
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

func (p *CLIPlanner) webStep(
	asset api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Web.Execute(asset, meta, configGroups, templateContext),
	}
}

func (p *CLIPlanner) dockerStep(asset api.DockerAsset, meta api.ReleaseMetadata) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Docker.Execute(asset, meta, p.watchProgress, asset.Dest),
	}
}

func (p *CLIPlanner) helmStep(
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Helm.Execute(asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) dockerLayerStep(
	asset api.DockerLayerAsset,
	metadata api.ReleaseMetadata,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.DockerLayer.Execute(asset, metadata, p.watchProgress),
	}
}

func (p *CLIPlanner) watchProgress(ch chan interface{}, debug log.Logger) error {
	var saveError error
	for msg := range ch {
		if msg == nil {
			continue
		}
		switch v := msg.(type) {
		case error:
			// continue reading on error to ensure channel is not blocked
			saveError = v
		case images.Progress:
			p.Daemon.SetProgress(config.JSONProgress("docker", v))
		case string:
			p.Daemon.SetProgress(config.StringProgress("docker", v))
		default:
			debug.Log("event", "progress", "message", fmt.Sprintf("%#v", v))
		}
	}
	return saveError
}
