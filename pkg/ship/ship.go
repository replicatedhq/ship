package ship

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/lifecycle"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	pkglogger "github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/replicatedcom/ship/pkg/version"
	"github.com/spf13/viper"
)

// repl ConfigOptionures an application
type Ship struct {
	Viper *viper.Viper

	Logger log.Logger

	APIPort  int
	Headless bool

	CustomerID     string
	ReleaseSemver  string
	ReleaseID      string
	ChannelID      string
	InstallationID string
	PlanOnly       bool

	Daemon     config.Daemon
	Resolver   *specs.Resolver
	StudioFile string
	Client     *specs.GraphQLClient
	UI         cli.Ui

	Runner *lifecycle.Runner
}

// FromViper gets an instance using viper to pull config
func FromViper(v *viper.Viper) (*Ship, error) {
	logger := pkglogger.FromViper(v)
	debug := level.Debug(log.With(logger, "phase", "ship.build", "source", "viper"))

	daemon := config.DaemonFromViper(v)

	debug.Log("event", "specresolver.build")
	resolver, err := specs.ResolverFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "get spec resolver")
	}

	debug.Log("event", "graphqlclient.build")
	graphql, err := specs.GraphQLClientFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "get graphql client")
	}

	debug.Log("event", "lifecycle.build")
	runner, err := lifecycle.RunnerFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "initialize lifecycle runner")
	}
	runner = runner.WithDaemon(daemon)

	debug.Log("event", "ui.build")
	return &Ship{
		Viper: v,

		Logger:   logger,
		Resolver: resolver,
		Client:   graphql,

		APIPort:  v.GetInt("api-port"),
		Headless: v.GetBool("headless"),

		CustomerID: v.GetString("customer-id"),

		ReleaseID:      v.GetString("release-id"),
		ReleaseSemver:  v.GetString("release-semver"),
		ChannelID:      v.GetString("channel-id"),
		InstallationID: v.GetString("installation-id"),
		StudioFile:     v.GetString("studio-file"),

		Daemon: daemon,
		UI:     ui.FromViper(v),
		Runner: runner,
	}, nil
}

func (s *Ship) shutdown(cancelFunc context.CancelFunc) {
	// need to pause beforce canceling the context, because we need
	// the daemon to stay up for a few seconds so the UI can know its
	// time to show the "You're all done" page
	level.Info(s.Logger).Log("event", "shutdown.prePause", "waitTime", "1s")
	time.Sleep(1 * time.Second)

	// now shut it all down, give things 5 seconds to clean up
	level.Info(s.Logger).Log("event", "shutdown.commence", "waitTime", "1s")
	cancelFunc()
	time.Sleep(1 * time.Second)

	level.Info(s.Logger).Log("event", "shutdown.complete")
}

// Execute starts ship
func (s *Ship) Execute(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.shutdown(cancelFunc)

	debug := level.Debug(log.With(s.Logger, "method", "execute"))

	debug.Log("method", "configure", "phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
		"customer-id", s.CustomerID,
		"installation-id", s.InstallationID,
		"plan_only", s.PlanOnly,
		"studio-file", s.StudioFile,
		"api-port", s.APIPort,
		"headless", s.Headless,
	)

	debug.Log("phase", "validate-inputs")

	if s.CustomerID == "" && s.StudioFile == "" {
		debug.Log("phase", "validate-inputs", "error", "missing customer ID")
		return errors.New("Missing parameter: customer-id. Please provide your license key or customer ID.")
	}

	if s.InstallationID == "" && s.StudioFile == "" {
		debug.Log("phase", "validate-inputs", "error", "missing installation ID")
		return errors.New("Missing parameter: installation-id. Please provide your license key or installation ID.")
	}

	debug.Log("phase", "validate-inputs", "status", "complete")

	selector := specs.Selector{
		CustomerID:     s.CustomerID,
		ReleaseSemver:  s.ReleaseSemver,
		ReleaseID:      s.ReleaseID,
		ChannelID:      s.ChannelID,
		InstallationID: s.InstallationID,
	}
	release, err := s.Resolver.ResolveRelease(ctx, selector)
	if err != nil {
		return errors.Wrap(err, "resolve specs")
	}

	runResultCh := make(chan error)
	go func() {
		defer close(runResultCh)
		err := s.Runner.Run(ctx, release)
		if err != nil {
			level.Error(s.Logger).Log("event", "shutdown", "reason", "error", "err", err)
		} else {
			level.Info(s.Logger).Log("event", "shutdown", "reason", "complete with no errors")
		}

		if err == nil {
			_ = s.Resolver.RegisterInstall(ctx, selector, release)
		}
		runResultCh <- err
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signalChan:
		level.Info(s.Logger).Log("event", "shutdown", "reason", "signal", "signal", sig)
		return nil
	case result := <-runResultCh:
		return result
	}
}

// ExitWithError should be called by the parent cobra commands if something goes wrong.
func (s *Ship) ExitWithError(err error) {
	if s.Viper.GetString("log-level") == "debug" {
		s.UI.Error(fmt.Sprintf("There was an unexpected error! %+v", err))
	} else {
		s.UI.Error(fmt.Sprintf("There was an unexpected error! %v", err))
	}
	s.UI.Output("")

	// TODO this should probably be part of lifecycle
	s.UI.Info("There was an error configuring the application. Please re-run with --log-level=debug and include the output in any support inquiries.")
	os.Exit(1)
}
