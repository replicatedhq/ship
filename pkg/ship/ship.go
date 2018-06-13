package ship

import (
	"context"
	"fmt"
	"os"
	"time"

	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/version"
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
	StateFile  string
	Client     *specs.GraphQLClient
	UI         cli.Ui

	Runner *lifecycle.Runner
}

// NewShip gets an instance using viper to pull config
func NewShip(
	logger log.Logger,
	v *viper.Viper,
	daemon config.Daemon,
	resolver *specs.Resolver,
	graphql *specs.GraphQLClient,
	runner *lifecycle.Runner,
	ui cli.Ui,
) (*Ship, error) {

	return &Ship{
		APIPort:        v.GetInt("api-port"),
		Headless:       v.GetBool("headless"),
		CustomerID:     v.GetString("customer-id"),
		ReleaseID:      v.GetString("release-id"),
		ReleaseSemver:  v.GetString("release-semver"),
		ChannelID:      v.GetString("channel-id"),
		InstallationID: v.GetString("installation-id"),
		StudioFile:     v.GetString("studio-file"),
		StateFile:      v.GetString("state-file"),

		Viper:    v,
		Logger:   logger,
		Resolver: resolver,
		Client:   graphql,
		Daemon:   daemon,
		UI:       ui,
		Runner:   runner.WithDaemon(daemon),
	}, nil
}

func (s *Ship) Shutdown(cancelFunc context.CancelFunc) {
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

// ExecuteAndMaybeExit runs ship to completion, and os.Exit()'s if it fails
func (s *Ship) ExecuteAndMaybeExit(ctx context.Context) {
	if err := s.Execute(ctx); err != nil {
		s.ExitWithError(err)
	}
}

func (s *Ship) Execute(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	debug := level.Debug(log.With(s.Logger, "method", "Execute"))

	debug.Log("method", "configure", "phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
		"customer-id", s.CustomerID,
		"installation-id", s.InstallationID,
		"plan_only", s.PlanOnly,
		"studio-file", s.StudioFile,
		"state-file", s.StateFile,
		"api-port", s.APIPort,
		"headless", s.Headless,
	)

	debug.Log("phase", "validate-inputs")

	if s.CustomerID == "" && s.StudioFile == "" {
		debug.Log("phase", "validate-inputs", "error", "missing customer ID")
		return errors.New("missing parameter customer-id, Please provide your license key or customer ID")
	}

	if s.InstallationID == "" && s.StudioFile == "" {
		debug.Log("phase", "validate-inputs", "error", "missing installation ID")
		return errors.New("missing parameter installation-id, Please provide your license key or customer ID")
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
		s.UI.Warn(fmt.Sprintf("%s received...", sig))
		return errors.Errorf("received signal %s", sig)
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
