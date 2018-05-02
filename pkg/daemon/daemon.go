package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	errInternal = errors.New("Internal Error")
)

// Daemon runs the ship api server.
type Daemon struct {
	CustomerID     string
	InstallationID string
	GraphQLClient  *specs.GraphQLClient
	UI             cli.Ui
	Logger         log.Logger
	Release        *api.Release
	Fs             afero.Afero
	Viper          *viper.Viper
}

// Serve starts the server with the given context
func (d *Daemon) Serve(ctx context.Context) error {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	g := gin.New()
	g.Use(cors.New(config))

	d.configureRoutes(g)

	addr := fmt.Sprintf(":%d", viper.GetInt("api-port"))
	server := http.Server{Addr: addr, Handler: g}
	errChan := make(chan error)

	go func() {
		errChan <- server.ListenAndServe()
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalChan:
		server.Shutdown(context.Background())
		level.Info(d.Logger).Log("event", "shutdown", "reason", "signal", "signal", sig)
		return nil
	case err := <-errChan:
		level.Error(d.Logger).Log("event", "shutdown", "reason", "errChan", "err", err)
		return err
	case <-ctx.Done():
		level.Error(d.Logger).Log("event", "shutdown", "reason", "context", "err", ctx.Err())
		return ctx.Err()
	}
}

func (d *Daemon) configureRoutes(g *gin.Engine) {
	root := g.Group("/")

	root.GET("/healthz", d.Healthz)
	root.GET("/metricz", d.Metricz)

	v1 := g.Group("/api/v1/config")
	v1.POST("live", d.postAppConfigLive)
	v1.PUT("", d.putAppConfig)
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

func (d *Daemon) postAppConfigLive(c *gin.Context) {
	// ItemValue is used as an unsaved (pending) value (copied from replicated appliance)
	type ItemValue struct {
		Name       string   `json:"name"`
		Value      string   `json:"value"`
		MultiValue []string `json:"multi_value"`
	}

	type Request struct {
		ItemValues []ItemValue `json:"item_values"`
	}

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

	resolver := &config.APIResolver{
		Logger:  d.Logger,
		Release: d.Release,
		Viper:   d.Viper,
	}

	state := state.StateManager{
		Logger: d.Logger,
	}
	savedState, err := state.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("msg", "failed to load state", "err", err)
		c.AbortWithStatus(500)
		return
	}

	for _, unsavedItemValue := range request.ItemValues {
		savedState[unsavedItemValue.Name] = unsavedItemValue.Value
	}

	resolvedConfig, err := resolver.ResolveConfig(c, nil, savedState)
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

	c.JSON(200, r)
}

func (d *Daemon) putAppConfig(c *gin.Context) {
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
	if err := stateManager.Serialize(nil, api.ReleaseMetadata{}, templateContext); err != nil {
		level.Error(d.Logger).Log("msg", "serialize state failed", "err", err)
		c.AbortWithStatus(500)
	}

	c.JSON(200, make(map[string]interface{}))
}
