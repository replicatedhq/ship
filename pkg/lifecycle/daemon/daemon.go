package daemon

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/version"
	"github.com/spf13/viper"
)

var (
	errInternal = errors.New("internal_error")
)

var _ daemontypes.Daemon = &ShipDaemon{}

// Daemon runs the ship api server.
type ShipDaemon struct {
	Logger       log.Logger
	WebUIFactory WebUIBuilder
	Viper        *viper.Viper
	// todo private this
	ExitChan  chan error
	StartOnce sync.Once

	*V1Routes
	*NavcycleRoutes
}

func (d *ShipDaemon) AwaitShutdown() error {
	return <-d.ExitChan
}

// "this is fine"
func (d *ShipDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {

	go d.StartOnce.Do(func() {
		err := d.Serve(ctx, release)
		level.Info(d.Logger).Log("event", "daemon.startonce.exit", err, "err")
		d.ExitChan <- err
	})

	return d.ExitChan
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
	} else {
		d.UI.Info(fmt.Sprintf(
			"\nPlease visit the following URL in your browser to continue the installation\n\n        %s\n\n ",
			openUrl,
		))
	}

	defer func() {
		debug.Log("event", "server.shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-errChan:
		level.Error(d.Logger).Log("event", "shutdown", "reason", "ExitChan", "err", err)
		return err
	case <-ctx.Done():
		level.Error(d.Logger).Log("event", "shutdown", "reason", "context", "err", ctx.Err())
		return ctx.Err()
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

	if d.V1Routes != nil {
		d.V1Routes.Register(root, release)
	}

	if d.NavcycleRoutes != nil {
		d.NavcycleRoutes.Register(root, release)
	}
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
