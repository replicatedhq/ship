package helmValues

import (
	"context"
	"path"
	"time"

	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type helmValues struct {
	Fs           afero.Afero
	Logger       log.Logger
	Daemon       daemontypes.Daemon
	StateManager state.Manager
}

func NewHelmValues(
	fs afero.Afero,
	logger log.Logger,
	daemon daemontypes.Daemon,
	stateManager state.Manager,
) lifecycle.HelmValues {
	return &helmValues{
		Fs:           fs,
		Logger:       logger,
		Daemon:       daemon,
		StateManager: stateManager,
	}
}

type daemonlessHelmValues struct {
	Fs           afero.Afero
	Logger       log.Logger
	StateManager state.Manager
}

func (d *daemonlessHelmValues) Execute(context.Context, *api.Release, *api.HelmValues) error {
	return d.resolveStateHelmValues()
}

func (h *helmValues) Execute(ctx context.Context, release *api.Release, step *api.HelmValues) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmValues"))

	daemonExitedChan := h.Daemon.EnsureStarted(ctx, release)

	debug.Log("event", "readfile.attempt", "dest", path.Join(constants.HelmChartPath, "values.yaml"))

	currentState, err := h.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	h.Daemon.SetProgress(daemontypes.StringProgress("helmValues", "generating installable application manifests"))

	h.Daemon.PushHelmValuesStep(ctx, daemontypes.HelmValues{
		Values: currentState.CurrentHelmValues(),
	}, daemon.HelmValuesActions())
	debug.Log("event", "step.pushed")

	return h.awaitContinue(ctx, daemonExitedChan)
}

func (h *helmValues) awaitContinue(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmValues", "awaitContinue"))
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-daemonExitedChan:
			if err != nil {
				return err
			}
			return errors.New("daemon exited")
		case <-h.Daemon.MessageConfirmedChan():
			debug.Log("message.confirmed")
			err := h.resolveStateHelmValues()
			if err != nil {
				return errors.Wrap(err, "resolve saved helm values from state.json")
			}
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}

func (h *helmValues) resolveStateHelmValues() error {
	return resolveStateHelmValues(h.Logger, h.StateManager, h.Fs)
}

func resolveStateHelmValues(logger log.Logger, manager state.Manager, fs afero.Afero) error {
	debug := level.Debug(log.With(logger, "step.type", "helmValues", "resolveHelmValues"))
	debug.Log("event", "tryLoadState")
	editState, err := manager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	helmValues := editState.CurrentHelmValues()
	if helmValues == "" {
		path := filepath.Join(constants.HelmChartPath, "values.yaml")
		bytes, err := fs.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read helm values from %s", constants.TempHelmValuesPath)
		}
		helmValues = string(bytes)
	}
	debug.Log("event", "tryLoadState")
	err = fs.MkdirAll(constants.TempHelmValuesPath, 0700)
	if err != nil {
		return errors.Wrapf(err, "make dir %s", constants.TempHelmValuesPath)
	}
	debug.Log("event", "writeTempValuesYaml")
	err = fs.WriteFile(path.Join(constants.TempHelmValuesPath, "values.yaml"), []byte(helmValues), 0644)
	if err != nil {
		return errors.Wrapf(err, "write values.yaml to %s", constants.TempHelmValuesPath)
	}
	return nil
}
