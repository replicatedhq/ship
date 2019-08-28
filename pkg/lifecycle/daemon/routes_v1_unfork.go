package daemon

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

func (d *V1Routes) UnforkSavedChan() chan interface{} {
	return d.UnforkSaved
}

func (d *V1Routes) PushUnforkStep(ctx context.Context, unfork daemontypes.Unfork) {
	debug := level.Debug(log.With(d.Logger, "method", "PushUnforkStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameUnfork
	d.currentStep = &daemontypes.Step{Unfork: &unfork}
	d.UnforkSaved = make(chan interface{}, 1)
}
