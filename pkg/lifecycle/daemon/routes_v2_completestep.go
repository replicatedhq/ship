package daemon

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
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
			return
		}

		// todo this will need to stream/push status
		if err := d.execute(step); err != nil {
			c.AbortWithError(500, err)
			return
		}

		newState := state.Versioned().WithCompletedStep(step)
		d.StateManager.Save(newState)
		c.JSON(200, map[string]interface{}{
			"status": "success",
		})
		return
	}

	d.errNotFond(c)
}

// temprorary home for a copy of pkg/lifecycle.StepExecutor while
// we re-implement each lifecycle step to not need a handle on a daemon
func (d *V2Routes) execute(step api.Step) error {
	debug := level.Debug(log.With(d.Logger, "method", "execute"))

	if step.Message != nil {
		debug.Log("event", "step.resolve", "type", "message")
		err := d.Messenger.Execute(context.Background(), d.Release, step.Message)
		debug.Log("event", "step.complete", "type", "message", "err", err)
		return errors.Wrap(err, "execute message step")
	} else if step.HelmIntro != nil {
		debug.Log("event", "step.resolve", "type", "helmIntro")
		err := d.HelmIntro.Execute(context.Background(), d.Release, step.HelmIntro)
		debug.Log("event", "step.complete", "type", "helmIntro", "err", err)
		return errors.Wrap(err, "execute helmIntro step")
	} else if step.Render != nil {
		debug.Log("event", "step.resolve", "type", "helmIntro")
		err := d.Renderer.Execute(context.Background(), d.Release, step.Render)
		debug.Log("event", "step.complete", "type", "helmIntro", "err", err)
		return errors.Wrap(err, "execute helmIntro step")
	}

	return errors.Errorf("unknown step %s:%s", step.ShortName(), step.Shared().ID)
}
