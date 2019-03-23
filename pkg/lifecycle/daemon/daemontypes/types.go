package daemontypes

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
)

type StatusReceiver interface {
	SetProgress(Progress)
	ClearProgress()
	PushStreamStep(context.Context, <-chan Message)
	SetStepName(context.Context, string)
	PushMessageStep(context.Context, Message, []Action)
}

// Daemon is a sort of UI interface. Some implementations start an API to
// power the on-prem web console. A headless implementation logs progress
// to stdout.
//
// A daemon is manipulated by lifecycle step handlers to present the
// correct UI to the user and collect necessary information
type Daemon interface {
	StatusReceiver

	EnsureStarted(context.Context, *api.Release) chan error
	PushRenderStep(context.Context, Render)
	PushHelmIntroStep(context.Context, HelmIntro, []Action)
	PushHelmValuesStep(context.Context, HelmValues, []Action)
	PushKustomizeStep(context.Context, Kustomize)
	AllStepsDone(context.Context)
	CleanPreviousStep()
	MessageConfirmedChan() chan string
	ConfigSavedChan() chan interface{}
	TerraformConfirmedChan() chan bool
	KustomizeSavedChan() chan interface{}
	UnforkSavedChan() chan interface{}
	GetCurrentConfig() map[string]interface{}
	AwaitShutdown() error
}

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
const StepNameUnfork = "unfork"

// the api abstraction for objects written in the YAML
// is starting to leak a little, so duplicating some stuff here
type Step struct {
	Source         api.Step        `json:"-"`
	Message        *Message        `json:"message,omitempty"`
	Render         *Render         `json:"render,omitempty"`
	HelmIntro      *HelmIntro      `json:"helmIntro,omitempty"`
	HelmValues     *HelmValues     `json:"helmValues,omitempty"`
	Kustomize      *Kustomize      `json:"kustomize,omitempty"`
	KustomizeIntro *KustomizeIntro `json:"kustomizeIntro,omitempty"`
	Unfork         *Unfork         `json:"unfork,omitempty"`
	Config         *Config         `json:"config,omitempty"`
}

// hack hack hack, I don't even know what to call this one
func NewStep(apiStep api.Step) Step {
	step := Step{
		Source: apiStep,
	}
	if apiStep.Message != nil {
		step.Message = &Message{
			Contents:    apiStep.Message.Contents,
			Level:       apiStep.Message.Level,
			TrustedHTML: true, // todo figure out trustedhtml
		}
	} else if apiStep.Render != nil {
		step.Render = &Render{}
	} else if apiStep.HelmIntro != nil {
		step.HelmIntro = &HelmIntro{
			IsUpdate: apiStep.HelmIntro.IsUpdate,
		}
	} else if apiStep.HelmValues != nil {
		step.HelmValues = &HelmValues{
			Values: "", // todo
			Path:   apiStep.HelmValues.Path,
		}
	} else if apiStep.Kustomize != nil {
		step.Kustomize = &Kustomize{
			BasePath: apiStep.Kustomize.Base,
		}
	} else if apiStep.KustomizeIntro != nil {
		step.KustomizeIntro = &KustomizeIntro{}
	} else if apiStep.Unfork != nil {
		step.Unfork = &Unfork{}
	} else if apiStep.Config != nil {
		step.Config = &Config{}
	}
	return step

}

type Message struct {
	Contents    string `json:"contents"`
	TrustedHTML bool   `json:"trusted_html"`
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
	IsUpdate bool `json:"isUpdate"`
}

type HelmValues struct {
	Values        string `json:"values"`
	DefaultValues string `json:"defaultValues"`
	ReleaseName   string `json:"helmName"`
	Namespace     string `json:"namespace"`
	Path          string `json:"path,omitempty" yaml:"path,omitempty" hcl:"path,omitempty"`
	// If set, Readme will override the default readme from the /metadata endpoint
	Readme string `json:"readme,omitempty" yaml:"readme,omitempty" hcl:"readme,omitempty"`
}

type Kustomize struct {
	BasePath string        `json:"basePath,omitempty"`
	Tree     filetree.Node `json:"tree,omitempty"`
}

type KustomizeIntro struct {
}

type Unfork struct {
}

type Config struct {
}
