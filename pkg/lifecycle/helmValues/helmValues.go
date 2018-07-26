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
	"github.com/spf13/afero"
)

type HelmValues interface {
	Execute(context.Context, *api.Release, *api.HelmValues) error
}

type helmValues struct {
	Fs     afero.Afero
	Logger log.Logger
	Daemon daemon.Daemon
}

func NewHelmValues(
	fs afero.Afero,
	logger log.Logger,
	daemon daemon.Daemon,
) HelmValues {
	return &helmValues{
		Fs:     fs,
		Logger: logger,
		Daemon: daemon,
	}
}

func (h *helmValues) Execute(ctx context.Context, release *api.Release, step *api.HelmValues) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmValues"))

	daemonExitedChan := h.Daemon.EnsureStarted(ctx, release)

	debug.Log("event", "readfile.attempt", "dest", path.Join(constants.BasePath, "values.yaml"))
	bytes, err := h.Fs.ReadFile(path.Join(constants.BasePath, "values.yaml"))
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
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}
