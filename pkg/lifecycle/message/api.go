package message

import (
	"context"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

type DaemonMessenger struct {
	dig.In
	Logger         log.Logger
	UI             cli.Ui
	Viper          *viper.Viper
	Daemon         daemontypes.Daemon
	BuilderBuilder *templates.BuilderBuilder
}

func (m *DaemonMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	debug := level.Debug(log.With(m.Logger, "struct", "daemonmessenger", "method", "execute"))

	daemonExitedChan := m.Daemon.EnsureStarted(ctx, release)

	builder, err := m.getBuilder(release.Metadata)
	if err != nil {
		return errors.Wrap(err, "get builder")
	}
	built, _ := builder.String(step.Contents)

	m.Daemon.PushMessageStep(ctx, daemontypes.Message{
		Contents: built,
		Level:    step.Level,
	}, daemon.MessageActions())

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
