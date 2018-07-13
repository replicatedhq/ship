package planner

import (
	"fmt"
	"path/filepath"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/images"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
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
		p.Daemon.SetProgress(daemon.JSONProgress("build", progress))

		if asset.Inline != nil {
			asset.Inline.Dest = filepath.Join(constants.InstallerPrefix, asset.Inline.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, p.inlineStep(*asset.Inline, configGroups, meta, templateContext))
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
		} else if asset.Terraform != nil {
			asset.Terraform.Dest = filepath.Join("installer", asset.Terraform.Dest)
			debug.Log("event", "asset.resolve", "asset.type", "terraform")
			plan = append(plan, p.terraformStep(*asset.Terraform, meta, templateContext, configGroups))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
			return nil, errors.New(
				"Unknown asset: type is not one of " +
					"[inline docker helm dockerlayer github terraform]",
			)
		}
	}
	return plan, nil
}

func (p *CLIPlanner) inlineStep(
	inline api.InlineAsset,
	configGroups []libyaml.ConfigGroup,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
		Execute:     p.Inline.Execute(inline, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) webStep(
	web api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        web.Dest,
		Description: web.Description,
		Execute:     p.Web.Execute(web, meta, configGroups, templateContext),
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

func (p *CLIPlanner) terraformStep(
	asset api.TerraformAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Terraform.Execute(asset, meta, configGroups, templateContext),
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
			p.Daemon.SetProgress(daemon.JSONProgress("docker", v))
		case string:
			p.Daemon.SetProgress(daemon.StringProgress("docker", v))
		default:
			debug.Log("event", "progress", "message", fmt.Sprintf("%#v", v))
		}
	}
	return saveError
}
