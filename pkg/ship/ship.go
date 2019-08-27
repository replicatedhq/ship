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
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helpers/flags"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/replicatedhq/ship/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// repl ConfigOptionures an application
type Ship struct {
	Viper *viper.Viper

	Logger log.Logger

	APIPort  int
	Headless bool
	Navcycle bool

	CustomerID     string
	ReleaseSemver  string
	InstallationID string
	PlanOnly       bool
	UploadAssetsTo string

	Daemon           daemontypes.Daemon
	Resolver         *specs.Resolver
	AppTypeInspector apptype.Inspector
	AppResolver      replicatedapp.Resolver
	Runbook          string
	UI               cli.Ui
	State            state.Manager
	IDPatcher        *specs.IDPatcher
	FS               afero.Afero
	Uploader         util.AssetUploader

	KustomizeRaw string
	Runner       *lifecycle.Runner
	StateManager state.Manager
}

// NewShip gets an instance using viper to pull config
func NewShip(
	logger log.Logger,
	v *viper.Viper,
	daemon daemontypes.Daemon,
	resolver *specs.Resolver,
	appresolver replicatedapp.Resolver,
	runner *lifecycle.Runner,
	ui cli.Ui,
	stateManager state.Manager,
	patcher *specs.IDPatcher,
	fs afero.Afero,
	inspector apptype.Inspector,
	uploader util.AssetUploader,
) (*Ship, error) {

	return &Ship{
		APIPort:        v.GetInt("api-port"),
		Headless:       v.GetBool("headless"),
		Navcycle:       v.GetBool("navcycle"),
		CustomerID:     v.GetString("customer-id"),
		ReleaseSemver:  v.GetString("release-semver"),
		InstallationID: v.GetString("installation-id"),
		UploadAssetsTo: v.GetString("upload-assets-to"),
		Runbook:        flags.GetCurrentOrDeprecatedString(v, "runbook", "studio-file"),

		KustomizeRaw: v.GetString("raw"),

		Viper:            v,
		Logger:           logger,
		Resolver:         resolver,
		AppResolver:      appresolver,
		AppTypeInspector: inspector,
		Daemon:           daemon,
		UI:               ui,
		Runner:           runner,
		State:            stateManager,
		IDPatcher:        patcher,
		FS:               fs,
		StateManager:     stateManager,
		Uploader:         uploader,
	}, nil
}

func (s *Ship) Shutdown(cancelFunc context.CancelFunc) {
	// remove the temp dir -- if we're exiting with an error, then cobra wont get a chance to clean up
	_ = s.FS.RemoveAll(constants.ShipPathInternalTmp)

	// need to pause before canceling the context, because we need
	// the daemon to stay up for a few seconds so the UI can know its
	// time to show the "You're all done" page
	level.Info(s.Logger).Log("event", "shutdown.prePause", "waitTime", "1s")
	time.Sleep(1 * time.Second)

	// now shut it all down, give things 1 second to clean up
	level.Info(s.Logger).Log("event", "shutdown.commence", "waitTime", "1s")
	cancelFunc()
	time.Sleep(1 * time.Second)

	level.Info(s.Logger).Log("event", "shutdown.complete")
}

// ExecuteAndMaybeExit runs ship to completion, and os.Exit()'s if it fails
func (s *Ship) ExecuteAndMaybeExit(ctx context.Context) error {
	if err := s.Execute(ctx); err != nil {
		s.ExitWithError(err)
		return err
	}
	return nil
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
		"runbook", s.Runbook,
		"api-port", s.APIPort,
		"headless", s.Headless,
	)

	debug.Log("phase", "validate-inputs")

	if s.CustomerID == "" && s.Runbook == "" && s.KustomizeRaw == "" {
		debug.Log("phase", "validate-inputs", "error", "missing customer ID")
		return errors.New("missing parameter customer-id, Please provide your license key or customer ID")
	}

	if s.InstallationID == "" && s.Runbook == "" && s.KustomizeRaw == "" {
		debug.Log("phase", "validate-inputs", "error", "missing installation ID")
		return errors.New("missing parameter installation-id, Please provide your license key or customer ID")
	}

	if err := s.maybeWriteStateFromFile(); err != nil {
		return err
	}

	s.State.UpdateVersion()

	debug.Log("phase", "validate-inputs", "status", "complete")

	selector := &replicatedapp.Selector{
		CustomerID:     s.CustomerID,
		ReleaseSemver:  s.ReleaseSemver,
		InstallationID: s.InstallationID,
	}

	localAppCopy := apptype.NewLocalAppCopy("replicated.app", "", "")
	release, err := s.AppResolver.ResolveAppRelease(ctx, selector, localAppCopy)
	if err != nil {
		return errors.Wrap(err, "resolve specs")
	}
	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)

	return s.execute(ctx, release, selector)
}

func (s *Ship) execute(ctx context.Context, release *api.Release, selector *replicatedapp.Selector) error {
	debug := level.Debug(log.With(s.Logger, "method", "execute"))
	warn := level.Debug(log.With(s.Logger, "method", "execute"))
	runResultCh := make(chan error)
	go func() {
		defer close(runResultCh)
		var err error
		// *wince* dex do this better
		if s.Headless {
			err = s.Runner.Run(ctx, release)
			s.Daemon.AllStepsDone(ctx)
		} else if s.Navcycle {
			s.Daemon.EnsureStarted(ctx, release)
			err = s.Daemon.AwaitShutdown()
		} else {
			err = s.Runner.Run(ctx, release)
			s.Daemon.AllStepsDone(ctx)
		}

		if err != nil {
			level.Error(s.Logger).Log("event", "shutdown", "reason", "error", "err", err)
		} else {
			level.Info(s.Logger).Log("event", "shutdown", "reason", "complete with no errors")
		}

		if err != nil {
			runResultCh <- err
			return
		}

		if selector != nil {
			_ = s.AppResolver.RegisterInstall(ctx, *selector, release)
		}

		if s.UploadAssetsTo != "" {
			debug.Log("event", "tmpdir.remove") // this feels like a weird place to do this. Open to other ways of excluding the tmp dir from the results
			err = os.RemoveAll(constants.ShipPathInternalTmp)
			if err != nil {
				warn.Log("event", "tmpdir.remove.fail", "path", constants.ShipPathInternalTmp, "err", err)
				// ignore, just do the upload anyway
			}
			debug.Log("event", "assets.upload")
			err = s.Uploader.UploadAssets(s.UploadAssetsTo)
		}
		runResultCh <- err
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signalChan:
		level.Info(s.Logger).Log("event", "shutdown", "reason", "signal", "signal", sig)
		s.UI.Warn(fmt.Sprintf("%s received...", sig))
		if sig == syscall.SIGINT {
			return nil
		}
		return errors.Errorf("received signal %q", sig)
	case result := <-runResultCh:
		return result
	}
}

func (s *Ship) maybeWriteStateFromFile() error {
	debug := level.Debug(log.With(s.Logger, "method", "maybeWriteStateFromFile"))

	// This is for integration tests to write the passed state.json to the correct path
	if s.Viper.GetString("state-from") != "file" {
		return nil
	}
	stateFilePath := s.Viper.GetString("state-file")
	if stateFilePath == "" {
		return nil
	}
	debug.Log("phase", "move", "state-file", stateFilePath)
	stateFile, err := s.FS.ReadFile(stateFilePath)
	if err != nil {
		return errors.Wrap(err, "read state-file")
	}
	if err := s.FS.WriteFile(constants.StatePath, stateFile, 0644); err != nil {
		return errors.Wrap(err, "write passed state file to constants.StatePath")
	}

	if err := s.State.ReloadFile(); err != nil {
		return errors.Wrap(err, "reload state-file")
	}

	return nil
}
