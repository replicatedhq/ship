package api

// A Lifecycle  is the top-level lifecycle object
type Lifecycle struct {
	V1 []Step `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// Step represents vendor-customized configuration steps & messaging
type Step struct {
	Message        *Message        `json:"message,omitempty" yaml:"message,omitempty" hcl:"message,omitempty"`
	Render         *Render         `json:"render,omitempty" yaml:"render,omitempty" hcl:"render,omitempty"`
	Terraform      *Terraform      `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	Kustomize      *Kustomize      `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
	KustomizeIntro *KustomizeIntro `json:"kustomizeIntro,omitempty" yaml:"kustomizeIntro,omitempty" hcl:"kustomizeIntro,omitempty"`
	KustomizeDiff  *KustomizeDiff  `json:"kustomizeDiff,omitempty" yaml:"kustomizeDiff,omitempty" hcl:"kustomizeDiff,omitempty"`
	HelmIntro      *HelmIntro      `json:"helmIntro,omitempty" yaml:"helmIntro,omitempty" hcl:"helmIntro,omitempty"`
	HelmValues     *HelmValues     `json:"helmValues,omitempty" yaml:"helmValues,omitempty" hcl:"helmValues,omitempty"`
}

func (s Step) Shared() *StepShared {
	if s.Message != nil {
		return &s.Message.StepShared
	} else if s.Render != nil {
		return &s.Render.StepShared
	} else if s.Terraform != nil {
		return &s.Terraform.StepShared
	} else if s.KustomizeIntro != nil {
		return &s.KustomizeIntro.StepShared
	} else if s.KustomizeDiff != nil {
		return &s.KustomizeDiff.StepShared
	} else if s.Kustomize != nil {
		return &s.Kustomize.StepShared
	} else if s.HelmIntro != nil {
		return &s.HelmIntro.StepShared
	} else if s.HelmValues != nil {
		return &s.HelmValues.StepShared
	}

	return nil
}

func (s Step) SetID(id string) {
	if s.Shared() != nil {
		s.Shared().ID = id
	}
}

func (s Step) ShortName() string {
	if s.Message != nil {
		return "message"
	} else if s.Render != nil {
		return "render"
	} else if s.Terraform != nil {
		return "terraform"
	} else if s.KustomizeIntro != nil {
		return "kustomizeIntro"
	} else if s.KustomizeDiff != nil {
		return "kustomizeDiff"
	} else if s.Kustomize != nil {
		return "kustomize"
	} else if s.HelmIntro != nil {
		return "helmIntro"
	} else if s.HelmValues != nil {
		return "helmValues"
	}
	return "step"
}

type StepShared struct {
	ID          string `json:"id,omitempty" yaml:"id,omitempty" hcl:",key"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" hcl:"description,omitempty"`
}

// Message is a lifeycle step to print a message
type Message struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Contents   string     `json:"contents" yaml:"contents" hcl:"contents"`
	Level      string     `json:"level,omitempty" yaml:"level,omitempty" hcl:"level,omitempty"`
}

// Render is a lifeycle step to collect config and render assets
type Render struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

// Terraform is a lifeycle step to execute `apply` for a runbook's terraform asset
type Terraform struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Path       string     `json:"path,omitempty" yaml:"path,omitempty" hcl:"path,omitempty"`
}

// Kustomize is a lifeycle step to generate overlays for generated assets.
// It does not take a kustomization.yml, rather it will generate one in the .ship/ folder
type Kustomize struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	BasePath   string     `json:"base_path,omitempty" yaml:"base_path,omitempty" hcl:"base_path,omitempty"`
	Dest       string     `json:"dest,omitempty" yaml:"dest,omitempty" hcl:"dest,omitempty"`
}

// KustomizeIntro is a lifeycle step to display an informative intro page for kustomize
type KustomizeIntro struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

// KustomizeDiff is a lifecycle step to display the diff of kustomized assets
type KustomizeDiff struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	BasePath   string     `json:"base_path,omitempty" yaml:"base_path,omitempty" hcl:"base_path,omitempty"`
	Dest       string     `json:"dest,omitempty" yaml:"dest,omitempty" hcl:"dest,omitempty"`
}

// HelmIntro is a lifecycle step to render persisted README.md in the .ship folder
type HelmIntro struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

// HelmValues is a lifecycle step to render persisted values.yaml in the .ship folder
// and save user input changes to values.yaml
type HelmValues struct {
	StepShared StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}
