package daemon

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
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
	step daemontypes.Message,
	actions []daemontypes.Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushMessageStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameMessage
	d.currentStep = &daemontypes.Step{Message: &step}
	d.currentStepActions = actions
}

func (d *V1Routes) PushStreamStep(
	ctx context.Context,
	msgs <-chan daemontypes.Message,
) {
	d.Lock()
	d.cleanPreviousStep()
	d.currentStepName = daemontypes.StepNameStream
	d.currentStep = &daemontypes.Step{Message: &daemontypes.Message{}}
	d.Unlock()

	for msg := range msgs {
		d.Lock()
		d.currentStep = &daemontypes.Step{Message: &msg}
		d.Unlock()
	}
}

func (d *V1Routes) TerraformConfirmedChan() chan bool {
	return d.TerraformConfirmed
}

func (d *V1Routes) PushRenderStep(
	ctx context.Context,
	step daemontypes.Render,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushRender"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameConfig
	d.currentStep = &daemontypes.Step{Render: &step}
}

func (d *V1Routes) PushHelmIntroStep(
	ctx context.Context,
	step daemontypes.HelmIntro,
	actions []daemontypes.Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmIntroStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameHelmIntro
	d.currentStep = &daemontypes.Step{HelmIntro: &step}
	d.currentStepActions = actions
}

func (d *V1Routes) PushHelmValuesStep(
	ctx context.Context,
	step daemontypes.HelmValues,
	actions []daemontypes.Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmValuesStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameHelmValues
	d.currentStep = &daemontypes.Step{HelmValues: &step}
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
