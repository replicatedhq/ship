package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"fmt"
)

// NavcycleRoutes provide workflow execution with standard browser navigation
type NavcycleRoutes struct {
	Logger       log.Logger
	TreeLoader   filetree.Loader
	StateManager state.Manager
	Messenger    lifecycle.Messenger
	HelmIntro    lifecycle.HelmIntro
	Renderer     lifecycle.Renderer
	Planner      planner.Planner
	StepExecutor V2Exectuor
	Fs           afero.Afero

	StepProgress *daemontypes.ProgressMap

	// This isn't known at injection time, so we have to set in Register
	Release *api.Release
}

// Register registers routes
func (d *NavcycleRoutes) Register(group *gin.RouterGroup, release *api.Release) {
	d.Release = release
	v1 := group.Group("/api/v1")
	v1.GET("/navcycle", d.getNavcycle)
	v1.GET("/navcycle/step/:step", d.getStep)
	v1.POST("/navcycle/step/:step", d.completeStep)
}

// returns false if aborted
func (d *NavcycleRoutes) maybeAbortDueToMissingRequirement(requires []string, c *gin.Context, requestedStepID string) (ok bool) {
	required, err := d.getRequiredButIncompleteStepFor(requires)
	if err != nil {
		c.AbortWithError(500, errors.Wrapf(err, "check requirements for step %s", requestedStepID))
		return false
	}
	if required != "" {
		d.errRequired(required, c)
		return false
	}
	return true
}

// this will return an incomplete step that is present in the list of required steps.
// if there are multiple required but incomplete steps, this will return the first one,
// although from a UI perspective the order is probably not strictly defined
func (d *NavcycleRoutes) getRequiredButIncompleteStepFor(requires []string) (string, error) {
	debug := level.Debug(log.With(d.Logger, "method", "getRequiredButIncompleteStepFor"))

	stepsCompleted := map[string]interface{}{}
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		return "", errors.Wrap(err, "load state")
	}
	if currentState.Versioned().V1.Lifecycle != nil &&
		currentState.Versioned().V1.Lifecycle.StepsCompleted != nil {
		stepsCompleted = currentState.Versioned().V1.Lifecycle.StepsCompleted
		debug.Log("event", "steps.notEmpty", "completed", fmt.Sprintf("%v", stepsCompleted))
	}

	for _, requiredStep := range requires {
		if _, ok := stepsCompleted[requiredStep]; ok {
			continue
		}
		debug.Log("event", "requiredStep.incomplete", "completed", stepsCompleted, "required", requiredStep)
		return requiredStep, nil
	}

	return "", nil
}

func (d *NavcycleRoutes) hydrateAndSend(step daemontypes.Step, c *gin.Context) {
	result, err := d.hydrateStep(step)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.JSON(200, result)
}

func (d *NavcycleRoutes) errRequired(required string, c *gin.Context) {
	c.JSON(400, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"requirementNotMet": map[string]interface{}{
				"required": required,
			},
		},
		"phase": "requirementNotMet",
	})
}

func (d *NavcycleRoutes) errNotFond(c *gin.Context) {
	c.JSON(404, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"notFound": map[string]interface{}{},
		},
		"phase": "notFound",
	})
}
