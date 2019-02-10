package planner

import (
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
)

func (p *CLIPlanner) githubStep(
	rootFs root.Fs,
	asset api.GitHubAsset,
	configGroups []libyaml.ConfigGroup,
	renderRoot string,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) Step {
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute:     p.GitHub.Execute(rootFs, asset, configGroups, renderRoot, meta, templateContext),
	}
}
