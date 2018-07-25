package daemon

const StepNameMessage = "message"
const StepNameConfig = "render.config"

// StepNameConfirm means that config is confirmed and assets are being rendered
const StepNameConfirm = "render.confirm"
const StepNamePlan = "terraform.plan"
const StepNameApply = "terraform.apply"
const StepNameReport = "terraform.report"

const StepNameKustomize = "kustomize"

// the api abstraction for objects written in the YAML
// is starting to leak a little, so duplicating some stuff here
type Step struct {
	Message   *Message   `json:"message"`
	Render    *Render    `json:"render"`
	Kustomize *Kustomize `json:"kustomize"`
}

type Message struct {
	Contents    string `json:"contents"`
	TrustedHTML bool   `json:"trusted_html"`
	Level       string `json:"level"`
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
	ButtonType  string        `json:"buttonType"`
	Text        string        `json:"text"`
	LoadingText string        `json:"loadingText"`
	OnClick     ActionRequest `json:"onclick"`
}

type Kustomize struct {
	Tree TreeNode `json:"tree"`
}

type TreeNode struct {
	Children   []TreeNode `json:"children"`
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	HasOverlay bool       `json:"hasOverlay"`
}
