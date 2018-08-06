package helmIntro

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/afero"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

type HelmIntro struct {
	Logger log.Logger
	Daemon daemon.Daemon
}

type DaemonlessHelmIntro struct {
	dig.In
	Logger log.Logger
}

func (d *DaemonlessHelmIntro) Execute(context.Context, *api.Release, *api.HelmIntro) error {
	level.Debug(d.Logger).Log("event", "DaemonlessHelmIntro.nothingToDo")
	return nil
}

func NewHelmIntro(
	v *viper.Viper,
	fs afero.Afero,
	logger log.Logger,
	daemon daemon.Daemon,
) lifecycle.HelmIntro {

	return &HelmIntro{
		Logger: logger,
		Daemon: daemon,
	}
}

func (h *HelmIntro) Execute(ctx context.Context, release *api.Release, step *api.HelmIntro) error {
	debug := level.Debug(log.With(h.Logger, "step.type", "helmIntro"))

	daemonExitedChan := h.Daemon.EnsureStarted(ctx, release)

	h.Daemon.PushHelmIntroStep(ctx, daemon.HelmIntro{}, daemon.HelmIntroActions())
	debug.Log("event", "step.pushed")

	return h.awaitContinue(ctx, daemonExitedChan)
}

func (h *HelmIntro) awaitContinue(ctx context.Context, daemonExitedChan chan error) error {
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
