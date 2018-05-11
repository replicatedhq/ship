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

// Daemon runs the ship api server.
type Daemon struct {
	Logger log.Logger
	Fs     afero.Afero
	Viper  *viper.Viper
	UI     cli.Ui

	sync.Mutex
	currentStep          *api.Step
	currentStepName      string
	currentStepConfirmed bool
	allStepsDone         bool
	pastSteps            []api.Step

	startOnce sync.Once

	initConfig    sync.Once
	ConfigSaved   chan interface{}
	CurrentConfig map[string]interface{}

	MessageConfirmed chan string

	//currentPlan planner.Plan
}

func (d *Daemon) PushStep(ctx context.Context, stepName string, step api.Step) {

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

func (d *Daemon) AllStepsDone(ctx context.Context) {
	d.Lock()
	defer d.Unlock()
	d.allStepsDone = true
}

// "this is fine"
func (d *Daemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {
	errChan := make(chan error)

	go d.startOnce.Do(func() {
		err := d.Serve(ctx, release)
		level.Info(d.Logger).Log("event", "daemon.startonce.exit", err, "err")
		errChan <- err
	})

	return errChan
}

// Serve starts the server with the given context
func (d *Daemon) Serve(ctx context.Context, release *api.Release) error {
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

func (d *Daemon) configureRoutes(g *gin.Engine, release *api.Release) {
	root := g.Group("/")

	root.GET("/healthz", d.Healthz)
	root.GET("/metricz", d.Metricz)
	v1 := g.Group("/api/v1")

	conf := v1.Group("/config")
	conf.POST("live", d.postAppConfigLive(release))
	conf.PUT("", d.putAppConfig)
	conf.PUT("finalize", d.finalizeAppConfig)

	life := v1.Group("/lifecycle")
	life.GET("current", d.getCurrentStep)
	life.GET("loading", d.getLoadingStep)

	mesg := v1.Group("/message")
	mesg.POST("confirm", d.postConfirmMessage)
	mesg.GET("get", d.getCurrentMessage)

	v1.GET("/channel", d.getChannel(release))

}

func (d *Daemon) getChannel(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"channelName": release.Metadata.ChannelName,
			"channelIcon": release.Metadata.ChannelIcon,
		})
	}

}

func (d *Daemon) getLoadingStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"loading": map[string]interface{}{},
		},
		"phase": "loading",
	})
}

func (d *Daemon) getDoneStep(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"currentStep": map[string]interface{}{
			"done": map[string]interface{}{},
		},
		"phase": "done",
	})
}

func (d *Daemon) getCurrentStep(c *gin.Context) {
	if d.currentStep == nil {
		d.getLoadingStep(c)
		return
	}
	if d.allStepsDone {
		d.getDoneStep(c)
		return
	}

	c.JSON(200, map[string]interface{}{
		"currentStep": d.currentStep,
		"phase":       d.currentStepName,
	})
}

func (d *Daemon) postConfirmMessage(c *gin.Context) {
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

func (d *Daemon) getCurrentMessage(c *gin.Context) {
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
func (d *Daemon) Healthz(c *gin.Context) {
	c.JSON(200, map[string]string{
		"version": os.Getenv("VERSION"),
	})
}

// Metricz returns (empty) metrics for this server
func (d *Daemon) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

func (d *Daemon) postAppConfigLive(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "postAppConfigLive"))

		if d.currentStepName != "render.config" {
			c.JSON(400, map[string]interface{}{
				"error": "no config step active",
			})
			return
		}

		// ItemValue is used as an unsaved (pending) value (copied from replicated appliance)
		type ItemValue struct {
			Name       string   `json:"name"`
			Value      string   `json:"value"`
			MultiValue []string `json:"multi_value"`
		}

		type Request struct {
			ItemValues []ItemValue `json:"item_values"`
		}

		debug.Log("event", "request.bind")
		var request Request
		if err := c.BindJSON(&request); err != nil {
			level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
			return
		}

		// TODO: handle multi value fields here
		itemValues := make(map[string]string)
		for _, itemValue := range request.ItemValues {
			if len(itemValue.MultiValue) > 0 {
				itemValues[itemValue.Name] = itemValue.MultiValue[0]
			} else {
				itemValues[itemValue.Name] = itemValue.Value
			}
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

		for _, unsavedItemValue := range request.ItemValues {
			savedStateMergedWithLiveValues[unsavedItemValue.Name] = unsavedItemValue.Value
		}

		resolver := &APIConfigRenderer{
			Logger: d.Logger,
			Viper:  d.Viper,
		}

		debug.Log("event", "getConfigForLiveRender")
		resolvedConfig, err := resolver.GetConfigForLiveRender(c, release, savedStateMergedWithLiveValues)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		type Result struct {
			Version int
			Groups  interface{}
		}
		r := Result{
			Version: 1,
			Groups:  resolvedConfig["config"],
		}

		debug.Log("event", "returnLiveConfig")
		c.JSON(200, r)
	}
}

func (d *Daemon) finalizeAppConfig(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "finalizeAppConfig"))
	d.putAppConfig(c)
	debug.Log("event", "configSaved.send.start")
	d.ConfigSaved <- nil
	debug.Log("event", "configSaved.send.complete")
}

func (d *Daemon) putAppConfig(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "putAppConfig"))
	d.Lock()
	defer d.Unlock()

	if d.currentStepName != "render.config" {
		c.JSON(400, map[string]interface{}{
			"error": "no config step active",
		})
		return
	}

	type Request struct {
		Options []struct {
			Name       string   `json:"name"`
			Value      string   `json:"value"`
			Data       string   `json:"data"`
			MultiValue []string `json:"multi_value"`
			MultiData  []string `json:"multi_data"`
		} `json:"options"`

		Validate bool `json:"validate"`
	}

	debug.Log("event", "request.bind")
	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		return
	}

	templateContext := make(map[string]interface{})
	for _, option := range request.Options {
		templateContext[option.Name] = option.Value
	}

	stateManager := state.StateManager{
		Logger: d.Logger,
	}
	debug.Log("event", "state.serialize")
	if err := stateManager.Serialize(nil, api.ReleaseMetadata{}, templateContext); err != nil {
		level.Error(d.Logger).Log("msg", "serialize state failed", "err", err)
		c.AbortWithStatus(500)
	}

	d.CurrentConfig = templateContext
	c.JSON(200, make(map[string]interface{}))
}

func (daemon *Daemon) NotifyStepChanged(stepType string) {
	// todo something with event streams
}
