package daemon

import (
	"context"

	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/statusonly"
)

func (d *V2Routes) completeStep(c *gin.Context) {
	requestedStep := c.Param("step")
	logger := log.With(d.Logger, "handler", "completeStep", "step", requestedStep)
	debug := level.Debug(logger)
	debug.Log()

	for _, step := range d.Release.Spec.Lifecycle.V1 {
		stepShared := step.Shared()
		stepID := stepShared.ID
		if stepID != requestedStep {
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

		// todo also stream statuses from execute step
		errChan := make(chan error)
		go func() {
			errChan <- d.StepExecutor(d, step)
		}()

		var async bool
		select {
		case err = <-errChan:
			if err != nil {
				debug.Log("event", "step.fail", "err", err)
				c.AbortWithError(500, err)
				return
			}
			debug.Log("event", "step.complete")

			// if it takes more than half a second, treat it as async, and provide status info
		case <-time.After(500 * time.Millisecond):
			debug.Log("event", "step.async")
			async = true
		}

		if async {
			c.JSON(200, map[string]interface{}{
				"status": "working",
				"phase":  step.ShortName(),
				"poll":   fmt.Sprintf("/lifecycle/step/%s", stepID),
			})
			go func() {
				err := d.awaitAsyncStep(errChan, debug, step)
				if err != nil {
					debug.Log("event", "execute.fail", "err", err)
					d.StepProgress.Store(stepID, daemontypes.StringProgress("v2router", "failed"))
					return
				}
				newState := state.Versioned().WithCompletedStep(step)

				err = d.StateManager.Save(newState)
				if err != nil {
					debug.Log("event", "state.save.fail", "err", err)
					return
				}
			}()
			return
		}
		level.Info(logger).Log("event", "task.complete", "progess", d.progress(step))
		d.StepProgress.Store(stepID, daemontypes.StringProgress("v2router", "failed"))
		newState := state.Versioned().WithCompletedStep(step)

		err = d.StateManager.Save(newState)
		if err != nil {
			debug.Log("event", "state.save.fail", "err", err)
			c.AbortWithError(500, errors.Wrap(err, "save state after successful execution"))
			return
		}

		c.JSON(200, map[string]interface{}{
			"status": "success",
			"phase":  step.ShortName(),
		})
		return
	}

	d.errNotFond(c)
}

func (d *V2Routes) awaitAsyncStep(errChan chan error, debug log.Logger, step api.Step) error {
	debug.Log("event", "async.await")
	for {
		select {
		// listen on err chan for step
		case err := <-errChan:
			if err != nil {
				level.Error(debug).Log("event", "async.fail", "err", err, "progress", d.progress(step))
				return err
			}
			level.Info(debug).Log("event", "task.complete", "progess", d.progress(step))
			return nil
		// debug log progress every ten seconds
		case <-time.After(10 * time.Second):
			debug.Log("event", "task.running", "progess", d.progress(step))
		}
	}
}

type V2Exectuor func(d *V2Routes, step api.Step) error

// temprorary home for a copy of pkg/lifecycle.StepExecutor while
// we re-implement each lifecycle step to not need a handle on a daemon (or something)
func (d *V2Routes) execute(step api.Step) error {
	debug := level.Debug(log.With(d.Logger, "method", "execute"))

	statusReceiver := &statusonly.StatusReceiver{
		OnProgress: func(progress daemontypes.Progress) {
			d.StepProgress.Store(step.Shared().ID, progress)
		},
	}
	planner := d.Planner.WithStatusReceiver(statusReceiver)
	renderer := d.Renderer.WithPlanner(planner)
	renderer = renderer.WithStatusReceiver(statusReceiver)

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

func (d *V2Routes) progress(step api.Step) daemontypes.Progress {
	progress, ok := d.StepProgress.Load(step.Shared().ID)
	if !ok {
		progress = daemontypes.StringProgress("v2router", "unknown")
	}
	return progress
}
