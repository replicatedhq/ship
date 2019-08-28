package planner

import (
	"fmt"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/images"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
)

type buildProgress struct {
	StepNumber int `json:"step_number"`
	TotalSteps int `json:"total_steps"`
}

// Build builds a plan in memory from assets+resolved config
func (p *CLIPlanner) Build(renderRoot string, assets []api.Asset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) (Plan, []string, error) {
	defer p.Status.ClearProgress()
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "phase", "plan"))

	debug.Log("renderRoot", renderRoot)
	rootFs := root.NewRootFS(p.Fs, renderRoot)

	builder, err := p.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
	if err != nil {
		return nil, nil, errors.Wrap(err, "init builder")
	}

	var plan Plan
	for i, asset := range assets {
		progress := buildProgress{
			StepNumber: i,
			TotalSteps: len(assets),
		}
		p.Status.SetProgress(daemontypes.JSONProgress("build", progress))

		if asset.Inline != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Inline.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "inline")
			if evaluatedWhen {
				plan = append(plan, p.inlineStep(rootFs, *asset.Inline, configGroups, meta, templateContext))
			}
		} else if asset.Docker != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Docker.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "docker")
			if evaluatedWhen {
				plan = append(plan, p.dockerStep(rootFs, *asset.Docker, meta, templateContext, configGroups))
			}
		} else if asset.Helm != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Helm.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "helm")
			if evaluatedWhen {
				plan = append(plan, p.helmStep(rootFs, *asset.Helm, meta, templateContext, configGroups))
			}
		} else if asset.Local != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Local.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "local")
			if evaluatedWhen {
				plan = append(plan, p.localStep(*asset.Local, meta, templateContext, configGroups))
			}
		} else if asset.DockerLayer != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.DockerLayer.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "dockerlayer")
			if evaluatedWhen {
				plan = append(plan, p.dockerLayerStep(rootFs, *asset.DockerLayer, meta, templateContext, configGroups))
			}
		} else if asset.Web != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Web.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "web")
			if evaluatedWhen {
				plan = append(plan, p.webStep(rootFs, *asset.Web, meta, configGroups, templateContext))
			}
		} else if asset.GitHub != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.GitHub.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "github")
			if evaluatedWhen {
				plan = append(plan, p.githubStep(rootFs, *asset.GitHub, configGroups, renderRoot, meta, templateContext))
			}
		} else if asset.Terraform != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.Terraform.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}

			p.logAssetResolve(debug, evaluatedWhen, "terraform")
			if evaluatedWhen {
				plan = append(plan, p.terraformStep(rootFs, *asset.Terraform, meta, templateContext, configGroups))
			}
		} else if asset.AmazonEKS != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.AmazonEKS.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}
			p.logAssetResolve(debug, evaluatedWhen, "amazon kubernetes cluster")
			if evaluatedWhen {
				plan = append(plan, p.amazonEKSStep(rootFs, *asset.AmazonEKS, meta, templateContext, configGroups))
			}
		} else if asset.GoogleGKE != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.GoogleGKE.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}
			p.logAssetResolve(debug, evaluatedWhen, "google kubernetes cluster")
			if evaluatedWhen {
				plan = append(plan, p.googleGKEStep(rootFs, *asset.GoogleGKE, meta, templateContext, configGroups))
			}
		} else if asset.AzureAKS != nil {
			evaluatedWhen, err := p.evalAssetWhen(debug, *builder, asset, asset.AzureAKS.AssetShared.When)
			if err != nil {
				return nil, nil, err
			}
			p.logAssetResolve(debug, evaluatedWhen, "azure kubernetes cluster")
			if evaluatedWhen {
				plan = append(plan, p.azureAKSStep(rootFs, *asset.AzureAKS, meta, templateContext, configGroups))
			}
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%#v", asset))
			return nil, nil, errors.New(
				"Unknown asset: type is not one of " +
					"[inline docker helm dockerlayer github terraform amazon_eks google_gke azure_aks]",
			)
		}
	}

	dests, err := planToDests(plan, builder)
	if err != nil {
		return nil, nil, err
	}

	return plan, dests, nil
}

func (p *CLIPlanner) inlineStep(
	rootFs root.Fs,
	inline api.InlineAsset,
	configGroups []libyaml.ConfigGroup,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        inline.Dest,
		Description: inline.Description,
		Execute:     p.Inline.Execute(rootFs, inline, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) webStep(
	rootFs root.Fs,
	web api.WebAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        web.Dest,
		Description: web.Description,
		Execute:     p.Web.Execute(rootFs, web, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) dockerStep(
	rootFs root.Fs,
	asset api.DockerAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: p.Docker.Execute(
			rootFs,
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
	rootFs root.Fs,
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Helm.Execute(rootFs, asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) localStep(
	asset api.LocalAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Local.Execute(asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) dockerLayerStep(
	rootFs root.Fs,
	asset api.DockerLayerAsset,
	metadata api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: p.DockerLayer.Execute(
			rootFs,
			asset,
			metadata,
			p.watchProgress,
			templateContext,
			configGroups,
		),
	}
}

func (p *CLIPlanner) terraformStep(
	rootFs root.Fs,
	asset api.TerraformAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.Terraform.Execute(rootFs, asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) amazonEKSStep(
	rootFs root.Fs,
	asset api.EKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.AmazonEKS.Execute(rootFs, asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) googleGKEStep(
	rootFs root.Fs,
	asset api.GKEAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.GoogleGKE.Execute(rootFs, asset, meta, templateContext, configGroups),
	}
}

func (p *CLIPlanner) azureAKSStep(
	rootFs root.Fs,
	asset api.AKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.AzureAKS.Execute(rootFs, asset, meta, templateContext, configGroups),
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
			p.Status.SetProgress(daemontypes.JSONProgress("docker", v))
		case string:
			p.Status.SetProgress(daemontypes.StringProgress("docker", v))
		default:
			debug.Log("event", "progress", "message", fmt.Sprintf("%#v", v))
		}
	}
	return saveError
}

func planToDests(plan Plan, builder *templates.Builder) ([]string, error) {
	var dests []string

	for _, step := range plan {
		dest, err := builder.String(step.Dest)
		if err != nil {
			return nil, errors.Wrapf(err, "building dest %q", step.Dest)
		}

		// special case for docker URL dests - don't attempt to remove url paths
		destinationURL, err := url.Parse(dest)
		// if there was an error parsing the dest as a url, or the scheme was not 'docker', add to the dests list as normal
		if err == nil && destinationURL.Scheme == "docker" {
			continue
		}

		dests = append(dests, dest)
	}

	return dests, nil
}
