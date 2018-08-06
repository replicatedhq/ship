package daemon

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/version"
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
	Logger       log.Logger
	WebUIFactory WebUIBuilder
	Viper        *viper.Viper
	exitChan     chan error
	startOnce    sync.Once

	*V1Routes
	*V2Routes
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
	if d.Viper.GetBool("no-open") {
		d.UI.Info(fmt.Sprintf(
			"\nPlease visit the following URL in your browser to continue the installation\n\n        %s\n\n ",
			openUrl,
		))
	} else {
		openBrowser, err := d.UI.Ask("Open browser to continue? (Y/n)")
		if err != nil {
			return err
		}

		openBrowser = strings.ToLower(strings.Trim(openBrowser, " \r\n"))
		if strings.Compare(openBrowser, "n") == 0 {
			d.UI.Info(fmt.Sprintf(
				"\nPlease visit the following URL in your browser to continue the installation\n\n        %s\n\n ",
				openUrl,
			))
		} else {
			err = d.OpenWebConsole(d.UI, openUrl)
			debug.Log("event", "console.open.fail.ignore", "err", err)
		}
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

	if d.V2Routes != nil {
		d.V2Routes.Register(root, release)
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
