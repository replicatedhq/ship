package api

// A Lifecycle  is the top-level lifecycle object
type Lifecycle struct {
	V1 []Step `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// Step represents vendor-customized configuration steps & messaging
type Step struct {
	Message *Message `json:"message,omitempty" yaml:"message,omitempty" hcl:"message,omitempty"`
	Render  *Render  `json:"render,omitempty" yaml:"render,omitempty" hcl:"render,omitempty"`
}

// Message is a lifeycle step to
type Message struct {
	Contents string `json:"contents" yaml:"contents" hcl:"contents"`
	Level    string `json:"level,omitempty" yaml:"level,omitempty" hcl:"level,omitempty"`
}

// Render is a lifeycle step to
type Render struct {
	SkipPlan         bool `json:"skip_plan" yaml:"skip_plan" hcl:"skip_plan"`
	SkipStateWarning bool `json:"skip_state_warning" yaml:"skip_state_warning" hcl:"skip_state_warning"`
}
