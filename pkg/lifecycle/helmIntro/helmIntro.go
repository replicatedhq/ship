package helmIntro

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/afero"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
)

type HelmIntro interface {
	Execute(context.Context, *api.Release, *api.HelmIntro) error
}

type helmIntro struct {
	Fs     afero.Afero
	Logger log.Logger
	Daemon daemon.Daemon
}

func NewHelmIntro(
	fs afero.Afero,
	logger log.Logger,
	daemon daemon.Daemon,
) HelmIntro {
	return &helmIntro{
		Fs:     fs,
		Logger: logger,
		Daemon: daemon,
	}
}

func (h *helmIntro) Execute(ctx context.Context, release *api.Release, step *api.HelmIntro) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmIntro"))

	daemonExitedChan := h.Daemon.EnsureStarted(ctx, release)

	h.Daemon.PushHelmIntroStep(ctx, daemon.HelmIntro{}, daemon.HelmIntroActions())
	debug.Log("event", "step.pushed")

	return h.awaitContinue(ctx, daemonExitedChan)
}

func (h *helmIntro) awaitContinue(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmIntro", "awaitContinue"))
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
