package planner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/templates"

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

	newConfigContext, err := p.BuilderBuilder.NewConfigContext(configGroups, templateContext)
	if err != nil {
		return nil, err
	}
	builder := p.BuilderBuilder.NewBuilder(
		p.BuilderBuilder.NewStaticContext(),
		newConfigContext,
	)

	var plan Plan
	for i, asset := range assets {
		progress := buildProgress{
			StepNumber: i,
			TotalSteps: len(assets),
		}
		p.Daemon.SetProgress(daemon.JSONProgress("build", progress))

		if asset.Inline != nil {
			asset.Inline.Dest = filepath.Join(constants.InstallerPrefix, asset.Inline.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.Inline.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "inline")
			if evaluatedWhen {
				plan = append(plan, p.inlineStep(*asset.Inline, configGroups, meta, templateContext))
			}
		} else if asset.Docker != nil {
			// TODO: Improve handling of docker scheme, this is done because config not parsed yet
			if !strings.HasPrefix(asset.Docker.Dest, "docker://") {
				asset.Docker.Dest = filepath.Join(constants.InstallerPrefix, asset.Docker.Dest)
			}
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.Docker.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "docker")
			if evaluatedWhen {
				plan = append(plan, p.dockerStep(*asset.Docker, meta, templateContext, configGroups))
			}
		} else if asset.Helm != nil {
			asset.Helm.Dest = filepath.Join(constants.InstallerPrefix, asset.Helm.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.Helm.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "helm")
			if evaluatedWhen {
				plan = append(plan, p.helmStep(*asset.Helm, meta, templateContext, configGroups))
			}
		} else if asset.DockerLayer != nil {
			asset.DockerLayer.Dest = filepath.Join(constants.InstallerPrefix, asset.DockerLayer.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.DockerLayer.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "dockerlayer")
			if evaluatedWhen {
				plan = append(plan, p.dockerLayerStep(*asset.DockerLayer, meta, templateContext, configGroups))
			}
		} else if asset.Web != nil {
			asset.Web.Dest = filepath.Join("installer", asset.Web.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.Web.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "web")
			if evaluatedWhen {
				plan = append(plan, p.webStep(*asset.Web, meta, configGroups, templateContext))
			}
		} else if asset.GitHub != nil {
			asset.GitHub.Dest = filepath.Join("installer", asset.GitHub.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.GitHub.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "github")
			if evaluatedWhen {
				plan = append(plan, p.githubStep(*asset.GitHub, configGroups, meta, templateContext))
			}
		} else if asset.Terraform != nil {
			asset.Terraform.Dest = filepath.Join("installer", asset.Terraform.Dest)
			evaluatedWhen, err := p.evalAssetWhen(debug, builder, asset, asset.Terraform.AssetShared.When)
			if err != nil {
				return nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "terraform")
			if evaluatedWhen {
				plan = append(plan, p.terraformStep(*asset.Terraform, meta, templateContext, configGroups))
			}
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

func (p *CLIPlanner) dockerStep(
	asset api.DockerAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: p.Docker.Execute(
			asset,
			meta,
			p.watchProgress,
			asset.Dest,
			templateContext,
			configGroups,
		),
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
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: p.DockerLayer.Execute(
			asset,
			metadata,
			p.watchProgress,
			templateContext,
			configGroups,
		),
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

func (p *CLIPlanner) evalAssetWhen(debug log.Logger, builder templates.Builder, asset api.Asset, assetWhen string) (bool, error) {
	builtWhen, err := builder.String(assetWhen)
	if err != nil {
		debug.Log("event", "asset.when.fail", "asset", fmt.Sprintf("%#v", asset))
		return false, err
	}

	builtWhenBool, err := builder.Bool(builtWhen, true)
	if err != nil {
		debug.Log("event", "asset.when.fail", "asset", fmt.Sprintf("%#v", asset))
		return false, err
	}

	return builtWhenBool, nil
}

func (p *CLIPlanner) logAssetResolve(debug log.Logger, when bool, assetType string) {
	if when {
		debug.Log("event", "asset.when.true", "asset.type", assetType)
		debug.Log("event", "asset.resolve", "asset.type", assetType)
	} else {
		debug.Log("event", "asset.when.false", "asset.type", assetType)
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
