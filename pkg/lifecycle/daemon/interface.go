package daemon

import (
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
)

const StepNameMessage = "message"
const StepNameConfig = "render.config"
const StepNameHelmIntro = "helm.intro"
const StepNameHelmValues = "helm.values"
const StepNameStream = "stream"

// StepNameConfirm means that config is confirmed and assets are being rendered
const StepNameConfirm = "render.confirm"
const StepNamePlan = "terraform.plan"
const StepNameApply = "terraform.apply"
const StepNameReport = "terraform.report"

const StepNameKustomize = "kustomize"

// the api abstraction for objects written in the YAML
// is starting to leak a little, so duplicating some stuff here
type Step struct {
	Source     api.Step    `json:"-"`
	Message    *Message    `json:"message,omitempty"`
	Render     *Render     `json:"render,omitempty"`
	HelmIntro  *HelmIntro  `json:"helmIntro,omitempty"`
	HelmValues *HelmValues `json:"helmValues,omitempty"`
	Kustomize  *Kustomize  `json:"kustomize,omitempty"`
}

func (s Step) Phase(isCurrent bool) string {
	return s.Source.ShortName()
}

type Message struct {
	Contents    string `json:"contents"`
	TrustedHTML bool   `json:"trusted_html"`
	Level       string `json:"level"`
}

type Render struct {
}

type StepResponse struct {
	CurrentStep Step      `json:"currentStep"`
	Phase       string    `json:"phase"`
	Actions     []Action  `json:"actions,omitempty"`
	Progress    *Progress `json:"progress,omitempty"`
}

type ActionRequest struct {
	URI    string `json:"uri"`
	Method string `json:"method"`
	Body   string `json:"body"`
}

type Action struct {
	Sort        int32         `json:"sort"`
	ButtonType  string        `json:"buttonType"`
	Text        string        `json:"text"`
	LoadingText string        `json:"loadingText"`
	OnClick     ActionRequest `json:"onclick"`
}

type HelmIntro struct {
}

type HelmValues struct {
	ID     string `json:"id"`
	Values string `json:"values"`
}

type Kustomize struct {
	ID       string        `json:"id"`
	BasePath string        `json:"basePath"`
	Tree     filetree.Node `json:"tree"`
}
