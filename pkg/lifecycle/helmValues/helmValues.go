package helmValues

import (
	"context"
	"path"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type HelmValues interface {
	Execute(context.Context, *api.Release, *api.HelmValues) error
}

type helmValues struct {
	Fs           afero.Afero
	Logger       log.Logger
	Daemon       daemon.Daemon
	StateManager state.Manager
}

func NewHelmValues(
	fs afero.Afero,
	logger log.Logger,
	daemon daemon.Daemon,
	stateManager state.Manager,
) HelmValues {
	return &helmValues{
		Fs:           fs,
		Logger:       logger,
		Daemon:       daemon,
		StateManager: stateManager,
	}
}

func (h *helmValues) Execute(ctx context.Context, release *api.Release, step *api.HelmValues) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmValues"))

	daemonExitedChan := h.Daemon.EnsureStarted(ctx, release)

	debug.Log("event", "readfile.attempt", "dest", path.Join(constants.KustomizeHelmPath, "values.yaml"))
	bytes, err := h.Fs.ReadFile(path.Join(constants.KustomizeHelmPath, "values.yaml"))
	if err != nil {
		return errors.Wrap(err, "read file values.yaml")
	}

	h.Daemon.PushHelmValuesStep(ctx, daemon.HelmValues{
		Values: string(bytes),
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
	debug := level.Debug(log.With(h.Logger, "step.type", "helmValues", "resolveHelmValues"))

	debug.Log("event", "tryLoadState")
	editState, err := h.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	helmValues := editState.CurrentHelmValues()

	debug.Log("event", "tryLoadState")
	err = h.Fs.MkdirAll(constants.TempHelmValuesPath, 0700)
	if err != nil {
		return errors.Wrapf(err, "make dir %s", constants.TempHelmValuesPath)
	}

	debug.Log("event", "writeTempValuesYaml")
	err = h.Fs.WriteFile(path.Join(constants.TempHelmValuesPath, "values.yaml"), []byte(helmValues), 0644)
	if err != nil {
		return errors.Wrapf(err, "write values.yaml to %s", constants.TempHelmValuesPath)
	}

	return nil
}
