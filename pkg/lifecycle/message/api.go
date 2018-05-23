package message

import (
	"context"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/spf13/viper"
)

type DaemonMessenger struct {
	Logger log.Logger
	UI     cli.Ui
	Viper  *viper.Viper
	Daemon config.Daemon
}

func (m *DaemonMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	debug := level.Debug(log.With(m.Logger, "struct", "daemonmessenger", "method", "execute"))

	daemonExitedChan := m.Daemon.EnsureStarted(ctx, release)

	builder := m.getBuilder(release.Metadata)
	built, _ := builder.String(step.Contents)

	m.Daemon.PushStep(ctx, "message", api.Step{
		Message: &api.Message{
			Contents: built,
			Level:    step.Level,
		}})
	debug.Log("event", "step.pushed")
	return m.awaitMessageConfirmed(ctx, daemonExitedChan)
}

func (m *DaemonMessenger) awaitMessageConfirmed(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(m.Logger, "struct", "daemonmessenger", "method", "message.confirm.await"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return ctx.Err()
		case err := <-daemonExitedChan:
			debug.Log("event", "daemon.exit")
			if err != nil {
				return err
			}
			return errors.New("daemon exited")
		case <-m.Daemon.MessageConfirmedChan():
			debug.Log("event", "message.confirmed")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}
