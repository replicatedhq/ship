package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"

	"github.com/replicatedhq/libyaml"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	errInternal = errors.New("internal_error")
)

type Daemon interface {
	EnsureStarted(context.Context, *api.Release) chan error
	PushStep(context.Context, string, api.Step)
	SetStepName(context.Context, string)
	AllStepsDone(context.Context)
	MessageConfirmedChan() chan string
	ConfigSavedChan() chan interface{}
	GetCurrentConfig() map[string]interface{}
	SetProgress(Progress)
	ClearProgress()
}

// Daemon runs the ship api server.
type ShipDaemon struct {
	Logger log.Logger
	Fs     afero.Afero
	Viper  *viper.Viper
	UI     cli.Ui

	sync.Mutex
	currentStep          *api.Step
	currentStepName      string
	currentStepConfirmed bool
	stepProgress         *Progress
	allStepsDone         bool
	pastSteps            []api.Step

	startOnce sync.Once

	initConfig    sync.Once
	ConfigSaved   chan interface{}
	CurrentConfig map[string]interface{}

	MessageConfirmed chan string

	//currentPlan planner.Plan
}

func (d *ShipDaemon) PushStep(ctx context.Context, stepName string, step api.Step) {

	d.Lock()
	defer d.Unlock()

	if d.currentStep != nil {
		d.pastSteps = append(d.pastSteps, *d.currentStep)
	}
	d.currentStepName = stepName
	d.currentStep = &step
	d.currentStepConfirmed = false
	d.NotifyStepChanged(stepName)
}

func (d *ShipDaemon) SetStepName(ctx context.Context, stepName string) {
	d.Lock()
	defer d.Unlock()
	d.currentStepName = stepName
}

func (d *ShipDaemon) AllStepsDone(ctx context.Context) {
	d.Lock()
	defer d.Unlock()
	d.allStepsDone = true
}

// "this is fine"
func (d *ShipDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {
	errChan := make(chan error)

	go d.startOnce.Do(func() {
		err := d.Serve(ctx, release)
		level.Info(d.Logger).Log("event", "daemon.startonce.exit", err, "err")
		errChan <- err
	})

	return errChan
}

// Serve starts the server with the given context
func (d *ShipDaemon) Serve(ctx context.Context, release *api.Release) error {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "method", "serve"))
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	g := gin.New()
	g.Use(cors.New(config))

	debug.Log("event", "routes.configure")
	d.configureRoutes(g, release)

	addr := fmt.Sprintf(":%d", viper.GetInt("api-port"))
	server := http.Server{Addr: addr, Handler: g}
	errChan := make(chan error)

	go func() {
		debug.Log("event", "server.listen", "server.addr", addr)
		errChan <- server.ListenAndServe()
	}()

	uiPort := 8025
	d.UI.Info(fmt.Sprintf(
		"Please visit the following URL in your browser to continue the installation\n\n        http://localhost:%d\n\n ",
		uiPort, // todo param this
	))

	defer func() {
		debug.Log("event", "server.shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-errChan:
		level.Error(d.Logger).Log("event", "shutdown", "reason", "errChan", "err", err)
		return err
	case <-ctx.Done():
		level.Error(d.Logger).Log("event", "shutdown", "reason", "context", "err", ctx.Err())
		return ctx.Err()
	}
}

func (d *ShipDaemon) configureRoutes(g *gin.Engine, release *api.Release) {
	root := g.Group("/")

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

	v1.GET("/channel", d.getChannel(release))

}

func (d *ShipDaemon) SetProgress(p Progress) {
	d.Lock()
	defer d.Unlock()
	d.stepProgress = &p
}

func (d *ShipDaemon) ClearProgress() {
	d.Lock()
	defer d.Unlock()
	d.stepProgress = nil
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

	result := map[string]interface{}{
		"currentStep": d.currentStep,
		"phase":       d.currentStepName,
	}
	if d.stepProgress != nil {
		result["progress"] = d.stepProgress
	}

	c.JSON(200, result)
}

func (d *ShipDaemon) postConfirmMessage(c *gin.Context) {
	d.Lock()
	defer d.Unlock()

	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "postConfirmMessage"))

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
	d.Lock()
	defer d.Unlock()

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
	c.JSON(200, map[string]string{
		"version": os.Getenv("VERSION"),
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
		debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "postAppConfigLive"))

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

		stateManager := state.StateManager{
			Logger: d.Logger,
		}
		debug.Log("event", "state.tryLoad")
		savedStateMergedWithLiveValues, err := stateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		for _, itemValue := range request.ItemValues {
			savedStateMergedWithLiveValues[itemValue.Name] = itemValue.Value
		}

		resolver := &APIConfigRenderer{
			Logger: d.Logger,
			Viper:  d.Viper,
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := resolver.ResolveConfig(c, release, savedStateMergedWithLiveValues)
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
		debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "finalizeAppConfig"))
		d.putAppConfig(release)(c)
		debug.Log("event", "configSaved.send.start")
		d.ConfigSaved <- nil
		debug.Log("event", "configSaved.send.complete")
	}
}

func (d *ShipDaemon) putAppConfig(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "putAppConfig"))
		d.Lock()
		defer d.Unlock()

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

		stateManager := state.StateManager{
			Logger: d.Logger,
		}
		debug.Log("event", "state.tryLoad")
		savedStateMergedWithLiveValues, err := stateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		for _, itemValue := range request.Options {
			savedStateMergedWithLiveValues[itemValue.Name] = itemValue.Value
		}

		resolver := &APIConfigRenderer{
			Logger: d.Logger,
			Viper:  d.Viper,
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := resolver.ResolveConfig(c, release, savedStateMergedWithLiveValues)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		validationErrors, err := resolver.ValidateConfig(c, release, resolvedConfig)
		if err != nil {
			level.Error(d.Logger).Log("event", "validateconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		} else if validationErrors != nil {
			// do something here and return parseable errors with 400
			c.AbortWithStatus(400)
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
		if err := stateManager.Serialize(nil, api.ReleaseMetadata{}, templateContext); err != nil {
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
