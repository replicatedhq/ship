package api

// A Lifecycle  is the top-level lifecycle object
type Lifecycle struct {
	V1 []Step `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// Step represents vendor-customized configuration steps & messaging
type Step struct {
	Message   *Message   `json:"message,omitempty" yaml:"message,omitempty" hcl:"message,omitempty"`
	Render    *Render    `json:"render,omitempty" yaml:"render,omitempty" hcl:"render,omitempty"`
	Terraform *Terraform `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	Kustomize *Kustomize `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
}

// Message is a lifeycle step to print a message
type Message struct {
	Contents string `json:"contents" yaml:"contents" hcl:"contents"`
	Level    string `json:"level,omitempty" yaml:"level,omitempty" hcl:"level,omitempty"`
}

// Render is a lifeycle step to collect config and render assets
type Render struct {
}

// Terraform is a lifeycle step to execute `apply` for a runbook's terraform asset
type Terraform struct {
}

// Kustomize is a lifeycle step to generate overlays for generated assets.
// It does not take a kustomization.yml, rather it will generate one in the .ship/ folder
type Kustomize struct {
	BasePath string `json:"base_path,omitempty" yaml:"base_path,omitempty" hcl:"base_path,omitempty"`
}
