package api

// Spec is the top level AppSpec document that defines an application
type Spec struct {
	Assets    Assets   `json:"assets" yaml:"assets" hcl:"asset"`
	Lifecycle Lifecyle `json:"lifecycle" yaml:"lifecycle" hcl:"lifecycle"`
	Config    Config   `json:"config" yaml:"config" hcl:"config"`
}
