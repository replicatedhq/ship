package ship

import (
	"context"

	"fmt"
	"os"

	"os/signal"
	"syscall"

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

	Daemon     *config.Daemon
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
	runner := lifecycle.RunnerFromViper(v).WithDaemon(daemon)

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

// Execute starts ship
func (s *Ship) Execute(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

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
		"studio", specs.AllowInlineSpecs,
		"api-port", s.APIPort,
		"headless", s.Headless,
	)

	debug.Log("phase", "validate-inputs")

	if s.StudioFile != "" && !specs.AllowInlineSpecs {
		debug.Log("phase", "validate-inputs", "error", "unsupported studio-file")
		return errors.New("unsupported configuration: studio-file")
	}

	if s.CustomerID == "" && s.StudioFile == "" {
		debug.Log("phase", "validate-inputs", "error", "missing customer ID")
		return errors.New("Missing parameter: customer-id. Please provide your license key or customer ID.")
	}
	debug.Log("phase", "validate-inputs", "status", "complete")

	release, err := s.Resolver.ResolveRelease(ctx, specs.Selector{
		CustomerID:     s.CustomerID,
		ReleaseSemver:  s.ReleaseSemver,
		ReleaseID:      s.ReleaseID,
		ChannelID:      s.ChannelID,
		InstallationID: s.InstallationID,
	})
	if err != nil {
		return errors.Wrap(err, "resolve specs")
	}

	go func() {
		err := s.Runner.Run(ctx, release)
		if err != nil {
			level.Error(s.Logger).Log("event", "shutdown", "reason", "error", "err", err)
		} else {
			level.Info(s.Logger).Log("event", "shutdown", "reason", "complete with no errors")
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan
	level.Info(s.Logger).Log("event", "shutdown", "reason", "signal", "signal", sig)
	return nil

	// todo send shipRegisterInstall mutation to pg.

	//dm := &daemon.Daemon{
	//	CustomerID:     s.CustomerID,
	//	InstallationID: s.InstallationID,
	//	GraphQLClient:  s.Client,
	//	UI:             s.UI,
	//	Logger:         s.Logger,
	//	Release:        release,
	//	Viper:          s.Viper,
	//}
	//
	//return errors.Wrap(dm.Serve(ctx), "run daemon")
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
