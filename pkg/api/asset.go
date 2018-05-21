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
	Inline *InlineAsset `json:"inline,omitempty" yaml:"inline,omitempty" hcl:"inline,omitempty"`
	Docker *DockerAsset `json:"docker,omitempty" yaml:"docker,omitempty" hcl:"docker,omitempty"`
	Github *GithubAsset `json:"github,omitempty" yaml:"github,omitempty" hcl:"github,omitempty"`
}

// InlineAsset is an asset whose contents are specified directly in the Spec
type InlineAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Contents    string `json:"contents" yaml:"contents" hcl:"contents"`
}

// DockerAsset is an asset whose contents are specified directly in the Spec
type DockerAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Image       string `json:"image" yaml:"image" hcl:"image"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
}

// GithubAsset is an asset whose contents are specified directly in the Spec
type GithubAsset struct {
	AssetShared `json:",inline" yaml:",inline" hcl:",inline"`
	Repo        string `json:"repo" yaml:"repo" hcl:"repo"`
	Ref         string `json:"ref" yaml:"ref" hcl:"ref"`
	Path        string `json:"path" yaml:"path" hcl:"path"`
	Source      string `json:"source" yaml:"source" hcl:"source"`
}
