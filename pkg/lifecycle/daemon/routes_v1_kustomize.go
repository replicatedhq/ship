package daemon

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

func (d *V1Routes) KustomizeSavedChan() chan interface{} {
	return d.KustomizeSaved
}

func (d *V1Routes) PushKustomizeStep(ctx context.Context, kustomize daemontypes.Kustomize) {
	debug := level.Debug(log.With(d.Logger, "method", "PushKustomizeStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = daemontypes.StepNameKustomize
	d.currentStep = &daemontypes.Step{Kustomize: &kustomize}
	d.KustomizeSaved = make(chan interface{}, 1)
}
