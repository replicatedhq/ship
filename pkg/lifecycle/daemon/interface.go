package daemon

const StepNameMessage = "message"
const StepNameConfig = "render.config"

// StepNameConfirm means that config is confirmed and assets are being rendered
const StepNameConfirm = "render.confirm"
const StepNamePlan = "terraform.plan"
const StepNameApply = "terraform.apply"
const StepNameReport = "terraform.report"

// the api abstraction for objects written in the YAML
// is starting to leak a little, so duplicating some stuff here
type Step struct {
	Message *Message `json:"message"`
	Render  *Render  `json:"render"`
}
type Message struct {
	Contents string `json:"contents"`
	Level    string `json:"level"`
}

type Render struct{}
