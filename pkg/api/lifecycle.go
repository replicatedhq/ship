package api

import "fmt"

// A Lifecycle  is the top-level lifecycle object
type Lifecycle struct {
	V1 []Step `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

// Step represents vendor-customized configuration steps & messaging
type Step struct {
	Message        *Message        `json:"message,omitempty" yaml:"message,omitempty" hcl:"message,omitempty"`
	Config         *ConfigStep     `json:"config,omitempty" yaml:"config,omitempty" hcl:"config,omitempty"`
	Render         *Render         `json:"render,omitempty" yaml:"render,omitempty" hcl:"render,omitempty"`
	Terraform      *Terraform      `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	Kustomize      *Kustomize      `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
	KustomizeIntro *KustomizeIntro `json:"kustomizeIntro,omitempty" yaml:"kustomizeIntro,omitempty" hcl:"kustomizeIntro,omitempty"`
	HelmIntro      *HelmIntro      `json:"helmIntro,omitempty" yaml:"helmIntro,omitempty" hcl:"helmIntro,omitempty"`
	HelmValues     *HelmValues     `json:"helmValues,omitempty" yaml:"helmValues,omitempty" hcl:"helmValues,omitempty"`
	KubectlApply   *KubectlApply   `json:"kubectl_apply,omitempty" yaml:"kubectl_apply,omitempty" hcl:"kubectl_apply,omitempty"`
}

func (s *Step) String() string {
	step := s.GetStep()
	return fmt.Sprintf("api.Step{ID: %q, Name: %q}", step.Shared().ID, step.ShortName())
}

type StepDetails interface {
	Shared() *StepShared
	ShortName() string
}

func (s Step) GetStep() StepDetails {
	if s.Message != nil {
		return s.Message

	} else if s.Render != nil {
		return s.Render
	} else if s.Config != nil {
		return s.Config
	} else if s.Terraform != nil {
		return s.Terraform
	} else if s.KustomizeIntro != nil {
		return s.KustomizeIntro
	} else if s.Kustomize != nil {
		return s.Kustomize
	} else if s.HelmIntro != nil {
		return s.HelmIntro
	} else if s.HelmValues != nil {
		return s.HelmValues
	}
	return nil
}
func (s Step) Shared() *StepShared { return s.GetStep().Shared() }
func (s Step) ShortName() string   { return s.GetStep().ShortName() }

type StepShared struct {
	ID          string   `json:"id,omitempty" yaml:"id,omitempty" hcl:",key"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty" hcl:"description,omitempty"`
	Requires    []string `json:"requires,omitempty" yaml:"requires,omitempty" hcl:"requires,omitempty"`
	Invalidates []string `json:"invalidates,omitempty" yaml:"invalidates,omitempty" hcl:"invalidates,omitempty"`
}

// Message is a lifeycle step to print a message
type Message struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Contents   string `json:"contents" yaml:"contents" hcl:"contents"`
	Level      string `json:"level,omitempty" yaml:"level,omitempty" hcl:"level,omitempty"`
}

func (m *Message) Shared() *StepShared { return &m.StepShared }
func (m *Message) ShortName() string   { return "message" }

// Render is a lifeycle step to collect config and render assets
type Render struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Root       string `json:"root,omitempty" yaml:"root,omitempty" hcl:"root,omitempty"`
}

func (r *Render) Shared() *StepShared { return &r.StepShared }

func (r *Render) ShortName() string { return "render" }

// Terraform is a lifeycle step to execute `apply` for a runbook's terraform asset
type Terraform struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Path       string `json:"path,omitempty" yaml:"path,omitempty" hcl:"path,omitempty"`
}

func (t *Terraform) Shared() *StepShared { return &t.StepShared }
func (t *Terraform) ShortName() string   { return "terraform" }

// Kustomize is a lifeycle step to generate overlays for generated assets.
// It does not take a kustomization.yml, rather it will generate one in the .ship/ folder
type Kustomize struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Base       string `json:"base,omitempty" yaml:"base,omitempty" hcl:"base,omitempty"`
	Dest       string `json:"dest,omitempty" yaml:"dest,omitempty" hcl:"dest,omitempty"`
	Overlay    string `json:"overlay,omitempty" yaml:"overlay,omitempty" hcl:"overlay,omitempty"`
}

func (k *Kustomize) OverlayPath() string {
	if k.Overlay == "" {
		return "overlays/ship"
	} else {
		return k.Overlay
	}
}

func (k *Kustomize) Shared() *StepShared { return &k.StepShared }
func (k *Kustomize) ShortName() string   { return "kustomize" }

// KustomizeIntro is a lifeycle step to display an informative intro page for kustomize
type KustomizeIntro struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

func (k *KustomizeIntro) Shared() *StepShared { return &k.StepShared }
func (k *KustomizeIntro) ShortName() string   { return "kustomize-intro" }

// HelmIntro is a lifecycle step to render persisted README.md in the .ship folder
type HelmIntro struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

func (h *HelmIntro) Shared() *StepShared { return &h.StepShared }
func (h *HelmIntro) ShortName() string   { return "helm-intro" }

// HelmValues is a lifecycle step to render persisted values.yaml in the .ship folder
// and save user input changes to values.yaml
type HelmValues struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

func (h *HelmValues) Shared() *StepShared { return &h.StepShared }
func (h *HelmValues) ShortName() string   { return "helm-values" }

type ConfigStep struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
}

func (c *ConfigStep) Shared() *StepShared {
	return &c.StepShared
}

func (c ConfigStep) ShortName() string {
	return "config"
}

// KubectlApply is a lifeycle step to execute `apply` for a kubeconfig asset
type KubectlApply struct {
	StepShared `json:",inline" yaml:",inline" hcl:",inline"`
	Path       string `json:"path,omitempty" yaml:"path,omitempty" hcl:"path,omitempty"`
	Kubeconfig string `json:"kubeconfig,omitempty" yaml:"kubeconfig,omitempty" hcl:"kubeconfig,omitempty"`
}

func (k *KubectlApply) Shared() *StepShared { return &k.StepShared }
func (k *KubectlApply) ShortName() string   { return "kubectl" }
