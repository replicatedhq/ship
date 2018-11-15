package daemon

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

func (d *V1Routes) requireKustomize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.currentStep == nil || d.currentStep.Kustomize == nil {
			c.AbortWithError(
				400,
				errors.Errorf("bad request: expected phase kustomize, was %q", d.currentStepName),
			)
			return
		}
		c.Next()

	}
}

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
