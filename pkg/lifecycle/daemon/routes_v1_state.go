package daemon

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func (d *V1Routes) locker(debug log.Logger) func() {
	debug.Log("event", "locker.try")
	d.Lock()
	debug.Log("event", "locker.acquired")

	return func() {
		d.Unlock()
		debug.Log("event", "locker.released")
	}
}

// resets previous step and prepares for new step.
// caller is responsible for locking the daemon before
// calling this
func (d *V1Routes) cleanPreviousStep() {
	if d.currentStep != nil {
		d.pastSteps = append(d.pastSteps, *d.currentStep)
	}
	d.currentStepName = ""
	d.currentStep = nil
	d.currentStepConfirmed = false
	d.currentStepActions = nil
}

func (d *V1Routes) CleanPreviousStep() {
	debug := level.Debug(log.With(d.Logger, "handler", "CleanPreviousStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()
}

func (d *V1Routes) PushMessageStep(
	ctx context.Context,
	step Message,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushMessageStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameMessage
	d.currentStep = &Step{Message: &step}
	d.currentStepActions = actions
}

func (d *V1Routes) PushStreamStep(
	ctx context.Context,
	msgs <-chan Message,
) {
	d.Lock()
	d.cleanPreviousStep()
	d.currentStepName = StepNameStream
	d.currentStep = &Step{Message: &Message{}}
	d.Unlock()

	for msg := range msgs {
		d.Lock()
		d.currentStep = &Step{Message: &msg}
		d.Unlock()
	}
}

func (d *V1Routes) TerraformConfirmedChan() chan bool {
	return d.TerraformConfirmed
}

func (d *V1Routes) PushRenderStep(
	ctx context.Context,
	step Render,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushRender"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameConfig
	d.currentStep = &Step{Render: &step}
}

func (d *V1Routes) PushHelmIntroStep(
	ctx context.Context,
	step HelmIntro,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmIntroStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameHelmIntro
	d.currentStep = &Step{HelmIntro: &step}
	d.currentStepActions = actions
}

func (d *V1Routes) PushHelmValuesStep(
	ctx context.Context,
	step HelmValues,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmValuesStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameHelmValues
	d.currentStep = &Step{HelmValues: &step}
	d.currentStepActions = actions
}

func (d *V1Routes) SetStepName(ctx context.Context, stepName string) {
	debug := level.Debug(log.With(d.Logger, "method", "SetStepName"))
	defer d.locker(debug)()
	d.currentStepName = stepName
}

func (d *V1Routes) AllStepsDone(ctx context.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "SetStepName"))
	defer d.locker(debug)()
	d.allStepsDone = true
}
