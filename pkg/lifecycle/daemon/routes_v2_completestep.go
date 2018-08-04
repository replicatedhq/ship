package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func (d *V2Routes) completeStep(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "getStep"))
	debug.Log()

	requestedStep := c.Param("step")

	for _, step := range d.Release.Spec.Lifecycle.V1 {
		stepShared := step.Shared()
		if stepShared.ID != requestedStep {
			continue
		}

		if ok := d.maybeAbortDueToMissingRequirement(
			stepShared.Requires,
			c,
			requestedStep,
		); !ok {
			return
		}

		state, err := d.StateManager.TryLoad()
		if err != nil {
			c.AbortWithError(500, err)
		}

		// mark step executing
		// execute step, stream status

		newState := state.Versioned().WithCompletedStep(step)
		d.StateManager.Save(newState)
	}

	d.errNotFond(c)
}
