package planner

import (
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
)

func (p *CLIPlanner) githubStep(
	asset api.GitHubAsset,
	configGroups []libyaml.ConfigGroup,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.GitHub.Execute(asset, configGroups, meta, templateContext),
	}
}
