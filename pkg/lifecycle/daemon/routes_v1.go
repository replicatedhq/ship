package daemon

import (
	"fmt"
	"net/http"
	"path"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/lint/rules"
	"k8s.io/helm/pkg/lint/support"
)

type V1Routes struct {
	Logger         log.Logger
	Fs             afero.Afero
	Viper          *viper.Viper
	UI             cli.Ui
	StateManager   state.Manager
	ConfigRenderer *resolve.APIConfigRenderer
	TreeLoader     filetree.Loader
	Patcher        patch.Patcher
	OpenWebConsole opener

	sync.Mutex
	currentStep          *daemontypes.Step
	currentStepName      string
	currentStepConfirmed bool
	stepProgress         *daemontypes.Progress
	allStepsDone         bool
	pastSteps            []daemontypes.Step

	// this is kind of kludged in,
	// it only makes sense for Message steps
	currentStepActions []daemontypes.Action

	ConfigSaved   chan interface{}
	CurrentConfig map[string]interface{}

	MessageConfirmed chan string

	TerraformConfirmed chan bool

	KustomizeSaved chan interface{}
	UnforkSaved    chan interface{}
	Release        *api.Release
}

func (d *V1Routes) Register(g *gin.RouterGroup, release *api.Release) {
	d.Release = release
	v1 := g.Group("/api/v1")

	life := v1.Group("/lifecycle")
	life.GET("current", d.getCurrentStep)
	life.GET("loading", d.getLoadingStep)

	mesg := v1.Group("/message")
	mesg.POST("confirm", d.postConfirmMessage)
	mesg.GET("get", d.getCurrentMessage)

	v1.POST("/helm-values", d.saveHelmValues)
}

func (d *V1Routes) SetProgress(p daemontypes.Progress) {
	defer d.locker(log.NewNopLogger())()
	d.stepProgress = &p
}

func (d *V1Routes) ClearProgress() {
	defer d.locker(log.With(log.NewNopLogger()))()
	d.stepProgress = nil
}

type SaveValuesRequest struct {
	Values      string `json:"values"`
	ReleaseName string `json:"releaseName"`
	Namespace   string `json:"namespace"`
}

func (d *V1Routes) saveHelmValues(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "saveHelmValues"))
	defer d.locker(debug)()
	var request SaveValuesRequest

	step, ok := d.getHelmValuesStepOrAbort(c)
	if !ok {
		return
	}

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		return
	}

	debug.Log("event", "validate")
	if ok := d.validateValuesOrAbort(c, request, *step); !ok {
		return
	}

	valuesPath := step.Path
	if valuesPath == "" {
		valuesPath = path.Join(constants.HelmChartPath, "values.yaml")
	}

	chartDefaultValues, err := d.Fs.ReadFile(valuesPath)
	if err != nil {

		level.Error(d.Logger).Log("event", "values.readDefault.fail", "err", err)
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "read file values.yaml"))
		return
	}

	debug.Log("event", "serialize.helmValues")
	if err := d.StateManager.SerializeHelmValues(request.Values, string(chartDefaultValues)); err != nil {
		debug.Log("event", "seralize.helmValues.fail", "err", err)
		c.AbortWithError(http.StatusInternalServerError, errInternal)
		return
	}

	debug.Log("event", "serialize.helmReleaseName")
	if len(request.ReleaseName) > 0 {
		if err := d.StateManager.SerializeReleaseName(request.ReleaseName); err != nil {
			debug.Log("event", "serialize.helmReleaseName.fail", "err", err)
			c.AbortWithError(http.StatusInternalServerError, errInternal)
			return
		}
	}
	if len(request.Namespace) > 0 {
		if err := d.StateManager.SerializeNamespace(request.Namespace); err != nil {
			debug.Log("event", "serialize.namespace.fail", "err", err)
			c.AbortWithError(http.StatusInternalServerError, errInternal)
			return
		}
	}
	c.String(http.StatusOK, "")
}

func (d *V1Routes) getHelmValuesStepOrAbort(c *gin.Context) (*daemontypes.HelmValues, bool) {
	// we don't support multiple values steps, but we could support multiple helm
	// values steps if client sent navcycle/step-id in this request
	for _, step := range d.Release.Spec.Lifecycle.V1 {
		if step.HelmValues != nil {
			return daemontypes.NewStep(step).HelmValues, true
		}
	}

	level.Warn(d.Logger).Log("event", "helm values step not found in lifecycle")
	c.JSON(http.StatusBadRequest, map[string]interface{}{
		"error": "no helm values step in lifecycle",
	})
	return nil, false
}

// validateValuesOrAbort checks the user-inputted helm values and will abort/bad request
// if invalid. Returns "false" if the request was aborted
func (d *V1Routes) validateValuesOrAbort(c *gin.Context, request SaveValuesRequest, step daemontypes.HelmValues) (ok bool) {
	debug := level.Debug(log.With(d.Logger, "handler", "validateValuesOrAbort"))

	fail := func(errors []string) bool {
		debug.Log(
			"event", "validate.fail",
			"errors", fmt.Sprintf("%+v", errors),
		)
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"errors": errors,
		})
		return false
	}

	// check we can read it (essentially a yaml validation)
	_, err := chartutil.ReadValues([]byte(request.Values))
	if err != nil {
		return fail([]string{err.Error()})
	}

	chartPath := constants.HelmChartPath
	if step.Path != "" {
		chartPath = path.Dir(step.Path)
	}

	// check that template functions like "required" are satisfied
	linter := support.Linter{ChartDir: chartPath}
	rules.Templates(&linter, []byte(request.Values), "", false)
	if len(linter.Messages) > 0 {
		var formattedErrors []string
		for _, message := range linter.Messages {
			formattedErrors = append(formattedErrors, message.Error())
		}
		return fail(formattedErrors)

	}
	return true
}

func (d *V1Routes) getLoadingStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"loading": map[string]interface{}{},
		},
		"phase": "loading",
	})
}

func (d *V1Routes) getDoneStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"done": map[string]interface{}{},
		},
		"phase": "done",
	})
}

func (d *V1Routes) getCurrentStep(c *gin.Context) {
	if d.currentStep == nil {
		d.getLoadingStep(c)
		return
	}
	if d.allStepsDone {
		d.getDoneStep(c)
		return
	}

	currentState, err := d.StateManager.CachedState()
	if err != nil {
		level.Error(d.Logger).Log("event", "tryLoad,fail", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}
	helmValues := currentState.CurrentHelmValues()
	if d.currentStep.HelmValues != nil && helmValues != "" {
		d.currentStep.HelmValues.Values = helmValues
	}

	result := daemontypes.StepResponse{
		CurrentStep: *d.currentStep,
		Phase:       d.currentStepName,
		Actions:     d.currentStepActions,
	}

	result.Progress = d.stepProgress

	c.JSON(200, result)
}

func (d *V1Routes) postConfirmMessage(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "postConfirmMessage"))
	defer d.locker(debug)()

	type Request struct {
		StepName string `json:"step_name"`
	}

	debug.Log("event", "request.bind")
	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		return
	}

	if d.currentStepName != request.StepName {
		c.JSON(400, map[string]interface{}{
			"error": "not current step",
		})
		return
	}

	if d.allStepsDone {
		c.JSON(400, map[string]interface{}{
			"error": "no more steps",
		})
		return
	}

	debug.Log("event", "confirm.step", "step", d.currentStepName)

	// Confirmation for each step will only be read once from the channel
	if d.currentStepConfirmed {
		c.String(200, "")
		return
	}

	d.currentStepConfirmed = true
	d.MessageConfirmed <- request.StepName

	c.String(200, "")
}

func (d *V1Routes) getCurrentMessage(c *gin.Context) {

	if d.currentStep == nil {
		c.JSON(400, map[string]interface{}{
			"error": "no steps taken",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"message": d.currentStep.Message,
	})
}
