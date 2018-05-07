package message

import (
	"context"

	"text/template"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/spf13/viper"
)

type DaemonMessenger struct {
	Logger             log.Logger
	UI                 cli.Ui
	Viper              *viper.Viper
	MaybeRunningDaemon *config.Daemon
}

func (m *DaemonMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	debug := level.Debug(log.With(m.Logger, "struct", "daemonmessenger", "method", "execute"))

	daemonExitedChan := m.MaybeRunningDaemon.EnsureStarted(ctx, release)

	m.MaybeRunningDaemon.PushStep(ctx, "message", api.Step{Message: step})
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
		case <-m.MaybeRunningDaemon.MessageConfirmed:
			debug.Log("event", "config.saved")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}

func (m *DaemonMessenger) funcMap() template.FuncMap {
	debug := level.Debug(log.With(m.Logger, "step.type", "render", "render.phase", "template"))

	configFunc := func(name string) interface{} {
		configItemValue := m.Viper.Get(name)
		if configItemValue == "" {
			debug.Log("event", "template.missing", "func", "config", "requested", name)
			return ""
		}
		return configItemValue
	}

	return map[string]interface{}{
		"config":       configFunc,
		"ConfigOption": configFunc,
		"context": func(name string) interface{} {
			switch name {
			case "state_file_path":
				return state.Path
			case "customer_id":
				return m.Viper.GetString("customer-id")
			}
			debug.Log("event", "template.missing", "func", "context", "requested", name)
			return ""
		},
	}
}
