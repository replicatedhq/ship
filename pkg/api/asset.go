package api

import "os"

// Assets is the top level assets object
type Assets struct {
	V1 []Asset `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// AssetShared is attributes common to all assets
type AssetShared struct {
	// Dest is where this file should be output
	Dest string `json:"dest" yaml:"dest" hcl:"dest"`
	// Mode is where this file should be output
	Mode os.FileMode `json:"mode" yaml:"mode" hcl:"mode"`
	// Description is an optional description
	Description string `json:"description" yaml:"description" hcl:"description"`
}

// Asset is a spec to generate one or more deployment assets
type Asset struct {
	Inline      *InlineAsset      `json:"inline,omitempty" yaml:"inline,omitempty" hcl:"inline,omitempty"`
	Docker      *DockerAsset      `json:"docker,omitempty" yaml:"docker,omitempty" hcl:"docker,omitempty"`
	DockerLayer *DockerLayerAsset `json:"dockerlayer,omitempty" yaml:"dockerlayer,omitempty" hcl:"dockerlayer,omitempty"`
	GitHub      *GitHubAsset      `json:"github,omitempty" yaml:"github,omitempty" hcl:"github,omitempty"`
	Web         *WebAsset         `json:"web,omitempty" yaml:"web,omitempty" hcl:"web,omitempty"`
	Helm        *HelmAsset        `json:"helm,omitempty" yaml:"helm,omitempty" hcl:"helm,omitempty"`
}

// InlineAsset is an asset whose contents are specified directly in the Spec
type InlineAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Contents    string `json:"contents" yaml:"contents" hcl:"contents"`
}

// DockerAsset is an asset that declares a docker image
type DockerAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Image       string `json:"image" yaml:"image" hcl:"image"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
}

// DockerLayerAsset is an asset that will unpack a docker layer at `dest`
type DockerLayerAsset struct {
	DockerAsset `json:",inline" yaml:",inline" hcl:",inline"`
	Layer       string `json:"layer" yaml:"layer" hcl:"layer"`
}

// GitHubAsset is an asset whose contents are specified directly in the Spec
type GitHubAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Repo        string `json:"repo" yaml:"repo" hcl:"repo"`
	Ref         string `json:"ref" yaml:"ref" hcl:"ref"`
	Path        string `json:"path" yaml:"path" hcl:"path"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
}

// WebAsset is an asset whose contents are specified by the HTML at the corresponding URL
type WebAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Body        string              `json:"body" yaml:"body" hcl:"body"`
	Headers     map[string][]string `json:"headers" yaml:"headers" hcl:"headers"`
	Method      string              `json:"method" yaml:"method" hcl:"method"`
	URL         string              `json:"url" yaml:"url" hcl:"url"`
}

// HelmAsset is an asset that declares a helm chart on github
type HelmAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Values      map[string]interface{} `json:"values" yaml:"values" hcl:"values"`
	HelmOpts    []string               `json:"helm_opts" yaml:"helm_opts" hcl:"helm_opts"`
	// GitHub references a github asset from which to pull the chart
	GitHub *GitHubAsset `json:"github" yaml:"github" hcl:"github"`
	// Local is an escape hatch, most impls will use github or some sort of ChartMuseum thing
	Local *LocalHelmOpts `json:"local,omitempty" yaml:"local,omitempty" hcl:"local,omitempty"`
}

// LocalHelmOpts specifies a helm chart that should be templated
// using other assets that are already present at `ChartRoot`
type LocalHelmOpts struct {
	ChartRoot string `json:"chart_root" yaml:"chart_root" hcl:"chart_root"`
}
