package daemon

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/contrib/static"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"

	"github.com/replicatedhq/libyaml"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	errInternal = errors.New("internal_error")
)

// Daemon is a sort of UI interface. Some implementations start an API to
// power the on-prem web console. A headless implementation logs progress
// to stdout.
//
// A daemon is manipulated by lifecycle step handlers to present the
// correct UI to the user and collect necessary information
type Daemon interface {
	EnsureStarted(context.Context, *api.Release) chan error
	PushMessageStep(context.Context, Message, []Action)
	PushStreamStep(context.Context, <-chan Message)
	PushRenderStep(context.Context, Render)
	PushHelmIntroStep(context.Context, HelmIntro, []Action)
	PushHelmValuesStep(context.Context, HelmValues, []Action)
	PushKustomizeStep(context.Context, Kustomize)
	SetStepName(context.Context, string)
	AllStepsDone(context.Context)
	CleanPreviousStep()
	MessageConfirmedChan() chan string
	ConfigSavedChan() chan interface{}
	TerraformConfirmedChan() chan bool
	KustomizeSavedChan() chan interface{}

	GetCurrentConfig() map[string]interface{}
	SetProgress(Progress)
	ClearProgress()
}

var _ Daemon = &ShipDaemon{}

// Daemon runs the ship api server.
type ShipDaemon struct {
	Logger         log.Logger
	Fs             afero.Afero
	Viper          *viper.Viper
	UI             cli.Ui
	StateManager   state.Manager
	ConfigRenderer *resolve.APIConfigRenderer
	WebUIFactory   WebUIBuilder
	TreeLoader     filetree.Loader
	OpenWebConsole opener

	sync.Mutex
	currentStep          *Step
	currentStepName      string
	currentStepConfirmed bool
	stepProgress         *Progress
	allStepsDone         bool
	pastSteps            []Step

	exitChan chan error

	// this is kind of kludged in,
	// it only makes sense for Message steps
	currentStepActions []Action

	startOnce sync.Once

	initConfig    sync.Once
	ConfigSaved   chan interface{}
	CurrentConfig map[string]interface{}

	MessageConfirmed chan string

	TerraformConfirmed chan bool

	KustomizeSaved chan interface{}
}

// resets previous step and prepares for new step.
// caller is responsible for locking the daemon before
// calling this
func (d *ShipDaemon) cleanPreviousStep() {
	if d.currentStep != nil {
		d.pastSteps = append(d.pastSteps, *d.currentStep)
	}
	d.currentStepName = ""
	d.currentStep = nil
	d.currentStepConfirmed = false
	d.currentStepActions = nil
}

func (d *ShipDaemon) CleanPreviousStep() {
	debug := level.Debug(log.With(d.Logger, "handler", "CleanPreviousStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()
}

func (d *ShipDaemon) PushMessageStep(
	ctx context.Context,
	step Message,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushMessageStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameMessage
	d.currentStep = &Step{Message: &step}
	d.currentStepActions = actions
	d.NotifyStepChanged(StepNameConfig)
}

func (d *ShipDaemon) PushStreamStep(
	ctx context.Context,
	msgs <-chan Message,
) {
	d.Lock()
	d.cleanPreviousStep()
	d.currentStepName = StepNameStream
	d.currentStep = &Step{Message: &Message{}}
	d.NotifyStepChanged(StepNameConfig)
	d.Unlock()

	for msg := range msgs {
		d.Lock()
		d.currentStep = &Step{Message: &msg}
		d.Unlock()
	}
}

func (d *ShipDaemon) TerraformConfirmedChan() chan bool {
	return d.TerraformConfirmed
}

func (d *ShipDaemon) PushRenderStep(
	ctx context.Context,
	step Render,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushRender"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameConfig
	d.currentStep = &Step{Render: &step}
	d.NotifyStepChanged(StepNameConfig)
}

func (d *ShipDaemon) PushHelmIntroStep(
	ctx context.Context,
	step HelmIntro,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmIntroStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameHelmIntro
	d.currentStep = &Step{HelmIntro: &step}
	d.currentStepActions = actions
	d.NotifyStepChanged(StepNameHelmIntro)
}

func (d *ShipDaemon) PushHelmValuesStep(
	ctx context.Context,
	step HelmValues,
	actions []Action,
) {
	debug := level.Debug(log.With(d.Logger, "handler", "PushHelmValuesStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameHelmValues
	d.currentStep = &Step{HelmValues: &step}
	d.currentStepActions = actions
	d.NotifyStepChanged(StepNameHelmValues)
}

func (d *ShipDaemon) SetStepName(ctx context.Context, stepName string) {
	debug := level.Debug(log.With(d.Logger, "method", "SetStepName"))
	defer d.locker(debug)()
	d.currentStepName = stepName
}

func (d *ShipDaemon) AllStepsDone(ctx context.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "SetStepName"))
	defer d.locker(debug)()
	d.allStepsDone = true
}

// "this is fine"
func (d *ShipDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {

	go d.startOnce.Do(func() {
		err := d.Serve(ctx, release)
		level.Info(d.Logger).Log("event", "daemon.startonce.exit", err, "err")
		d.exitChan <- err
	})

	return d.exitChan
}

// Serve starts the server with the given context
func (d *ShipDaemon) Serve(ctx context.Context, release *api.Release) error {
	debug := level.Debug(log.With(d.Logger, "method", "serve"))
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	g := gin.New()
	g.Use(cors.New(config))

	debug.Log("event", "routes.configure")
	d.configureRoutes(g, release)

	apiPort := viper.GetInt("api-port")
	addr := fmt.Sprintf(":%d", apiPort)
	server := http.Server{Addr: addr, Handler: g}
	errChan := make(chan error)

	go func() {
		debug.Log("event", "server.listen", "server.addr", addr)
		errChan <- server.ListenAndServe()
	}()

	openUrl := fmt.Sprintf("http://localhost:%d", apiPort)
	if !d.Viper.GetBool("no-open") {
		err := d.OpenWebConsole(d.UI, openUrl)
		debug.Log("event", "console.open.fail.ignore", "err", err)
	}

	defer func() {
		debug.Log("event", "server.shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-errChan:
		level.Error(d.Logger).Log("event", "shutdown", "reason", "exitChan", "err", err)
		return err
	case <-ctx.Done():
		level.Error(d.Logger).Log("event", "shutdown", "reason", "context", "err", ctx.Err())
		return ctx.Err()
	}
}

func (d *ShipDaemon) locker(debug log.Logger) func() {
	debug.Log("event", "locker.try")
	d.Lock()
	debug.Log("event", "locker.acquired")

	return func() {
		d.Unlock()
		debug.Log("event", "locker.released")
	}
}

func (d *ShipDaemon) configureRoutes(g *gin.Engine, release *api.Release) {

	root := g.Group("/")
	g.Use(static.Serve("/", d.WebUIFactory("dist")))
	g.NoRoute(func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "NoRoute"))
		debug.Log("event", "not found")
		if c.Request.URL != nil {
			path := c.Request.URL.Path
			static.Serve(path, d.WebUIFactory("dist"))(c)

		}
		static.Serve("/", d.WebUIFactory("dist"))(c)
	})

	root.GET("/healthz", d.Healthz)
	root.GET("/metricz", d.Metricz)
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
	v1.POST("/kustomize/patch", d.requireKustomize(), d.createMergePatch)
}

func (d *ShipDaemon) createMergePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "createMergePatch"))
	type Request struct {
		Original string `json:"original"`
		Modified string `json:"modified"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
	}

	patch, err := d.createTwoWayMergePatch(request.Original, request.Modified)
	if err != nil {
		level.Error(d.Logger).Log("event", "create two way merge patch", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	c.JSON(200, map[string]interface{}{
		"patch": string(patch),
	})
}

func (d *ShipDaemon) SetProgress(p Progress) {
	defer d.locker(log.NewNopLogger())()
	d.stepProgress = &p
}

func (d *ShipDaemon) ClearProgress() {
	defer d.locker(log.With(log.NewNopLogger()))()
	d.stepProgress = nil
}

func (d *ShipDaemon) getHelmMetadata(release *api.Release) gin.HandlerFunc {
	debug := level.Debug(log.With(d.Logger, "handler", "getHelmMetadata"))
	debug.Log("event", "response.metadata")
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"metadata": release.Metadata.HelmChartMetadata,
		})
	}
}

func (d *ShipDaemon) saveHelmValues(c *gin.Context) {
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

	debug.Log("event", "serialize.helmValues")
	err := d.StateManager.SerializeHelmValues(request.Values)
	if err != nil {
		level.Error(d.Logger).Log("event", "seralize.helmValues.fail", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}
	c.String(200, "")
}

func (d *ShipDaemon) getChannel(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"channelName": release.Metadata.ChannelName,
			"channelIcon": release.Metadata.ChannelIcon,
		})
	}

}

func (d *ShipDaemon) getLoadingStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"loading": map[string]interface{}{},
		},
		"phase": "loading",
	})
}

func (d *ShipDaemon) getDoneStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"done": map[string]interface{}{},
		},
		"phase": "done",
	})
}

func (d *ShipDaemon) getCurrentStep(c *gin.Context) {
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

	state, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "tryLoad,fail", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}
	helmValues := state.CurrentHelmValues()
	if d.currentStep.HelmValues != nil && helmValues != "" {
		d.currentStep.HelmValues.Values = helmValues
	}

	result := StepResponse{
		CurrentStep: *d.currentStep,
		Phase:       d.currentStepName,
		Actions:     d.currentStepActions,
	}

	result.Progress = d.stepProgress

	c.JSON(200, result)
}

// todo load the tree and any overlays, but fake it for now

func (d *ShipDaemon) terraformApply(c *gin.Context) {
	debug := log.With(level.Debug(d.Logger), "handler", "terraformApply")
	defer d.locker(debug)()
	debug.Log("event", "terraform.apply.send")
	d.TerraformConfirmed <- true
	debug.Log("event", "terraform.apply.sent")
}

func (d *ShipDaemon) terraformSkip(c *gin.Context) {
	debug := log.With(level.Debug(d.Logger), "handler", "terraformSkip")
	defer d.locker(debug)()
	debug.Log("event", "terraform.skip.send")
	d.TerraformConfirmed <- false
	debug.Log("event", "terraform.skip.sent")
}

func (d *ShipDaemon) postConfirmMessage(c *gin.Context) {
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

func (d *ShipDaemon) getCurrentMessage(c *gin.Context) {

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

// Healthz returns a 200 with the version if provided
func (d *ShipDaemon) Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func (d *ShipDaemon) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

type ConfigOption struct {
	Name       string   `json:"name"`
	Value      string   `json:"value"`
	Data       string   `json:"data"`
	MultiValue []string `json:"multi_value"`
	MultiData  []string `json:"multi_data"`
}

func (d *ShipDaemon) postAppConfigLive(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "postAppConfigLive"))

		if d.currentStepName != StepNameConfig {
			c.JSON(400, map[string]interface{}{
				"error": "no config step active",
			})
			return
		}

		type Request struct {
			ItemValues []ConfigOption `json:"item_values"`
		}

		debug.Log("event", "request.bind")
		var request Request
		if err := c.BindJSON(&request); err != nil {
			level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
			return
		}

		debug.Log("event", "state.tryLoad")
		savedSate, err := d.StateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		liveValues := make(map[string]interface{})
		for _, itemValue := range request.ItemValues {
			liveValues[itemValue.Name] = itemValue.Value
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := d.ConfigRenderer.ResolveConfig(c, release, savedSate.CurrentConfig(), liveValues, true)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		type Result struct {
			Version int
			Groups  []libyaml.ConfigGroup
		}
		r := Result{
			Version: 1,
			Groups:  resolvedConfig,
		}

		debug.Log("event", "returnLiveConfig")
		c.JSON(200, r)
	}
}

func (d *ShipDaemon) finalizeAppConfig(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "finalizeAppConfig"))
		d.putAppConfig(release)(c)
		debug.Log("event", "configSaved.send.start")
		d.ConfigSaved <- nil
		debug.Log("event", "configSaved.send.complete")
	}
}

func (d *ShipDaemon) putAppConfig(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "putAppConfig"))
		defer d.locker(debug)()

		if d.currentStepName != StepNameConfig {
			c.JSON(400, map[string]interface{}{
				"error": "no config step active",
			})
			return
		}

		type Request struct {
			Options  []ConfigOption `json:"options"`
			Validate bool           `json:"validate"`
		}

		debug.Log("event", "request.bind")
		var request Request
		if err := c.BindJSON(&request); err != nil {
			level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
			return
		}

		debug.Log("event", "state.tryLoad")
		savedState, err := d.StateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		liveValues := make(map[string]interface{})
		for _, itemValue := range request.Options {
			liveValues[itemValue.Name] = itemValue.Value
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := d.ConfigRenderer.ResolveConfig(c, release, savedState.CurrentConfig(), liveValues, false)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		if validationErrors := resolve.ValidateConfig(resolvedConfig); validationErrors != nil {
			c.AbortWithStatusJSON(400, validationErrors)
			return
		}

		// NOTE: what about multi value, data, multi data?
		templateContext := make(map[string]interface{})
		for _, configGroup := range resolvedConfig {
			for _, configItem := range configGroup.Items {
				templateContext[configItem.Name] = configItem.Value
			}
		}

		debug.Log("event", "state.serialize")
		if err := d.StateManager.SerializeConfig(nil, api.ReleaseMetadata{}, templateContext); err != nil {
			level.Error(d.Logger).Log("msg", "serialize state failed", "err", err)
			c.AbortWithStatus(500)
		}

		d.CurrentConfig = templateContext
		c.JSON(200, make(map[string]interface{}))
	}
}

func (d *ShipDaemon) MessageConfirmedChan() chan string {
	return d.MessageConfirmed
}

func (d *ShipDaemon) ConfigSavedChan() chan interface{} {
	return d.ConfigSaved
}

func (d *ShipDaemon) GetCurrentConfig() map[string]interface{} {
	if d.CurrentConfig == nil {
		return make(map[string]interface{})
	}
	return d.CurrentConfig
}

func (d *ShipDaemon) NotifyStepChanged(stepType string) {
	// todo something with event streams
}
