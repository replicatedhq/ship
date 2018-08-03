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

// hack hack hack, I don't even know what to call this one
func NewStep(apiStep api.Step) Step {
	step := Step{Source: apiStep}
	if apiStep.Message != nil {
		step.Message = &Message{
			Contents:    apiStep.Message.Contents,
			Level:       apiStep.Message.Level,
			TrustedHTML: true, // todo figure out trustedhtml
		}
	} else if apiStep.Render != nil {
		step.Render = &Render{}
	} else if apiStep.HelmIntro != nil {
		step.HelmIntro = &HelmIntro{}
	} else if apiStep.HelmValues != nil {
		step.HelmValues = &HelmValues{
			Values: "", // todo
		}
	} else if apiStep.Kustomize != nil {
		step.Kustomize = &Kustomize{
			BasePath: apiStep.Kustomize.BasePath,
		}
	}
	return step

}

type Message struct {
	Contents    string `json:"contents"`
	TrustedHTML bool   `json:"trusted_html,omitempty"`
	Level       string `json:"level,omitempty"`
}

type Render struct{}

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
	Sort        int32         `json:"sort,omitempty"`
	ButtonType  string        `json:"buttonType,omitempty"`
	Text        string        `json:"text,omitempty"`
	LoadingText string        `json:"loadingText,omitempty"`
	OnClick     ActionRequest `json:"onclick,omitempty"`
}

type HelmIntro struct {
}

type HelmValues struct {
	Values string `json:"values"`
}

type Kustomize struct {
	BasePath string        `json:"basePath,omitempty"`
	Tree     filetree.Node `json:"tree,omitempty"`
}
