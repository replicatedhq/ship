package daemon

import (
	"fmt"
	"net/http"
	"sync"

	"k8s.io/helm/pkg/lint/rules"
	"k8s.io/helm/pkg/lint/support"

	"github.com/replicatedhq/ship/pkg/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
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

	initConfig    sync.Once
	ConfigSaved   chan interface{}
	CurrentConfig map[string]interface{}

	MessageConfirmed chan string

	TerraformConfirmed chan bool

	KustomizeSaved chan interface{}
}

func (d *V1Routes) Register(g *gin.RouterGroup, release *api.Release) {
	v1 := g.Group("/api/v1")
	conf := v1.Group("/config")
	conf.POST("live", d.postAppConfigLive(release))
	conf.PUT("", d.putAppConfig(release))
	conf.PUT("finalize", d.finalizeAppConfig(release))

	life := v1.Group("/lifecycle")
	life.GET("current", d.getCurrentStep)
	life.GET("loading", d.getLoadingStep)

	mesg := v1.Group("/message")
	mesg.POST("confirm", d.postConfirmMessage)
	mesg.GET("get", d.getCurrentMessage)

	tf := v1.Group("/terraform")
	tf.POST("apply", d.terraformApply)
	tf.POST("skip", d.terraformSkip)

	v1.GET("/channel", d.getChannel(release))

	v1.GET("/helm-metadata", d.getHelmMetadata(release))
	v1.POST("/helm-values", d.saveHelmValues)

	v1.POST("/kustomize/file", d.requireKustomize(), d.kustomizeGetFile)
	v1.POST("/kustomize/save", d.requireKustomize(), d.kustomizeSaveOverlay)
	v1.POST("/kustomize/finalize", d.requireKustomize(), d.kustomizeFinalize)
	v1.POST("/kustomize/patch", d.requireKustomize(), d.createOrMergePatch)
	v1.DELETE("/kustomize/patch", d.requireKustomize(), d.deletePatch)
	v1.POST("/kustomize/apply", d.requireKustomize(), d.applyPatch)
}

func (d *V1Routes) applyPatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "applyPatch"))
	type Request struct {
		Patch string `json:"patch"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	modified, err := d.Patcher.ApplyPatch(request.Patch)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to merge patch with base", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	c.JSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}

func (d *V1Routes) createOrMergePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "createOrMergePatch"))
	type Request struct {
		Original string `json:"original"`
		Modified string `json:"modified"`
		Current  string `json:"current"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	debug.Log("event", "load.originalFile")
	original, err := d.TreeLoader.LoadFile(d.currentStep.Kustomize.BasePath, request.Original)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to read original file", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	patch, err := d.Patcher.CreateTwoWayMergePatch(original, request.Modified)
	if err != nil {
		level.Error(d.Logger).Log("event", "create two way merge patch", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	if request.Current != "" {
		out, err := d.Patcher.MergePatches([]byte(request.Current), patch)
		if err != nil {
			level.Error(d.Logger).Log("event", "merge current and new patch", "err", err)
			c.AbortWithError(500, errors.New("internal_server_error"))
		}
		c.JSON(200, map[string]interface{}{
			"patch": string(out),
		})
	} else {
		c.JSON(200, map[string]interface{}{
			"patch": string(patch),
		})
	}
}

func (d *V1Routes) deletePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deletePatch"))
	pathQueryParam := c.Query("path")
	if pathQueryParam == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
	}

	debug.Log("event")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "try load state failed", "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		level.Error(d.Logger).Log("event", "empty kustomize")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	shipOverlay := kustomize.Ship()
	if len(shipOverlay.Patches) == 0 {
		level.Error(d.Logger).Log("event", "empty ship overlay")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	_, ok := shipOverlay.Patches[pathQueryParam]
	if !ok {
		level.Error(d.Logger).Log("event", "patch does not exist")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	debug.Log("event", "deletePatch", "path", pathQueryParam)
	delete(shipOverlay.Patches, pathQueryParam)

	if err := d.StateManager.SaveKustomize(kustomize); err != nil {
		level.Error(d.Logger).Log("event", "patch does not exist")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}

func (d *V1Routes) SetProgress(p daemontypes.Progress) {
	defer d.locker(log.NewNopLogger())()
	d.stepProgress = &p
}

func (d *V1Routes) ClearProgress() {
	defer d.locker(log.With(log.NewNopLogger()))()
	d.stepProgress = nil
}

func (d *V1Routes) getHelmMetadata(release *api.Release) gin.HandlerFunc {
	debug := level.Debug(log.With(d.Logger, "handler", "getHelmMetadata"))
	debug.Log("event", "response.metadata")
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"metadata": release.Metadata.HelmChartMetadata,
		})
	}
}

func (d *V1Routes) saveHelmValues(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "saveHelmValues"))
	defer d.locker(debug)()
	type Request struct {
		Values string `json:"values"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
	}

	debug.Log("event", "validate")
	linter := support.Linter{ChartDir: constants.KustomizeHelmPath}
	rules.Templates(&linter, []byte(request.Values), "", false)

	if len(linter.Messages) > 0 {
		var formattedErrors []string
		for _, message := range linter.Messages {
			formattedErrors = append(formattedErrors, message.Error())
		}

		debug.Log(
			"event", "validate.fail",
			"errors", fmt.Sprintf("%+v", formattedErrors),
		)
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"errors": formattedErrors,
		})
		return
	}

	debug.Log("event", "serialize")
	err := d.StateManager.SerializeHelmValues(request.Values)
	if err != nil {
		debug.Log("event", "seralize.fail", "err", err)
		c.AbortWithError(http.StatusInternalServerError, errors.New("internal_server_error"))
	}
	c.String(http.StatusOK, "")
}

func (d *V1Routes) getChannel(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"channelName": release.Metadata.ChannelName,
			"channelIcon": release.Metadata.ChannelIcon,
		})
	}

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

	// checking non-nil instead of step name
	if d.currentStep.Kustomize != nil {
		tree, err := d.loadKustomizeTree()
		if err != nil {
			level.Error(d.Logger).Log("event", "loadTree.fail", "err", err)
			c.AbortWithError(500, errors.New("internal_server_error"))
			return
		}
		d.currentStep.Kustomize.Tree = *tree
	}

	currentState, err := d.StateManager.TryLoad()
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

func (d *V1Routes) terraformApply(c *gin.Context) {
	debug := log.With(level.Debug(d.Logger), "handler", "terraformApply")
	defer d.locker(debug)()
	debug.Log("event", "terraform.apply.send")
	d.TerraformConfirmed <- true
	debug.Log("event", "terraform.apply.sent")
}

func (d *V1Routes) terraformSkip(c *gin.Context) {
	debug := log.With(level.Debug(d.Logger), "handler", "terraformSkip")
	defer d.locker(debug)()
	debug.Log("event", "terraform.skip.send")
	d.TerraformConfirmed <- false
	debug.Log("event", "terraform.skip.sent")
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
	return
}
