package daemon

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

// NavcycleRoutes provide workflow execution with standard browser navigation
type NavcycleRoutes struct {
	Logger       log.Logger
	TreeLoader   filetree.Loader
	StateManager state.Manager
	StepExecutor V2Executor
	Fs           afero.Afero
	Shutdown     chan interface{}

	StepProgress *daemontypes.ProgressMap

	Messenger      lifecycle.Messenger
	HelmIntro      lifecycle.HelmIntro
	HelmValues     lifecycle.HelmValues
	Kustomizer     lifecycle.Kustomizer
	KustomizeIntro lifecycle.KustomizeIntro
	Renderer       lifecycle.Renderer
	Terraformer    lifecycle.Terraformer
	Planner        planner.Planner
	Patcher        patch.Patcher
	ConfigRenderer *resolve.APIConfigRenderer
	KubectlApply   lifecycle.KubectlApply

	ConfigSaved        chan interface{}
	TerraformConfirmed chan bool
	CurrentConfig      map[string]interface{}

	KubectlConfirmed chan bool

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
	v1.POST("/shutdown", d.shutdown)

	kustom := v1.Group("/kustomize")
	kustom.POST("file", d.kustomizeGetFile)
	kustom.POST("save", d.kustomizeSaveOverlay)
	kustom.POST("finalize", d.kustomizeFinalize)
	kustom.POST("patch", d.createOrMergePatch)
	kustom.DELETE("patch", d.deletePatch)
	kustom.POST("apply", d.applyPatch)

	conf := v1.Group("/config")
	conf.POST("live", d.postAppConfigLive(release))
	conf.PUT("", d.putAppConfig(release))
	conf.PUT("finalize", d.finalizeAppConfig(release))

	terr := v1.Group("/terraform")
	terr.POST("apply", d.terraformApply)
	terr.POST("skip", d.terraformApply)

	kube := v1.Group("/kubectl")
	kube.POST("confirm", d.kubectlConfirm)
}

func (d *NavcycleRoutes) shutdown(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "shutdown"))

	debug.Log("event", "shutdownFromUI")
	d.Shutdown <- nil
	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "shutdown",
	})
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

func (d *NavcycleRoutes) errNotFound(c *gin.Context) {
	c.JSON(404, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"notFound": map[string]interface{}{},
		},
		"phase": "notFound",
	})
}
