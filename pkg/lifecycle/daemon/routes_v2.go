package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/state"
)

type V2Routes struct {
	Logger       log.Logger
	TreeLoader   filetree.Loader
	StateManager state.Manager
	// This isn't known at injection time, so we have to set in Register
	Release *api.Release
}

func (d *V2Routes) Register(group *gin.RouterGroup, release *api.Release) {
	d.Release = release
	v2 := group.Group("/api/v2")
	v2.GET("/lifecycle", d.getLifecycle)
	v2.GET("/lifecycle/step/:step", d.getStep)
	v2.POST("/lifecycle/step/:step", d.completeStep)
}

// returns false if aborted
func (d *V2Routes) maybeAbortDueToMissingRequirement(requires []string, c *gin.Context, requestedStepID string) (ok bool) {
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

func (d *V2Routes) getRequiredButIncompleteStepFor(requires []string) (string, error) {
	debug := level.Debug(log.With(d.Logger, "method", "getRequiredButIncompleteStepFor"))

	stepsCompleted := map[string]interface{}{}
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		return "", errors.Wrap(err, "load state")
	}
	if currentState.Versioned().V1.Lifecycle != nil &&
		currentState.Versioned().V1.Lifecycle.StepsCompleted != nil {
		stepsCompleted = currentState.Versioned().V1.Lifecycle.StepsCompleted
		debug.Log("event", "steps.notEmpty", "completed", stepsCompleted)
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

func (d *V2Routes) hydrateAndSend(step Step, c *gin.Context) {
	result, err := d.hydrateStep(step, true)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.JSON(200, result)
}

func (d *V2Routes) errRequired(required string, c *gin.Context) {
	c.JSON(400, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"requirementNotMet": map[string]interface{}{
				"required": required,
			},
		},
		"phase": "requirementNotMet",
	})
}

func (d *V2Routes) errNotFond(c *gin.Context) {
	c.JSON(404, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"notFound": map[string]interface{}{},
		},
		"phase": "notFound",
	})
}
