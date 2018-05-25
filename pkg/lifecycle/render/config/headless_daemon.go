package config

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
)

type HeadlessDaemon struct {
	StateManager *state.StateManager
	Logger       log.Logger
	UI           cli.Ui
}

func (d *HeadlessDaemon) EnsureStarted(context.Context, *api.Release) chan error {
	return make(chan error)
}

func (d *HeadlessDaemon) PushStep(context.Context, string, api.Step) {}

func (d *HeadlessDaemon) SetStepName(context.Context, string) {}

func (d *HeadlessDaemon) AllStepsDone(context.Context) {}

func (d *HeadlessDaemon) MessageConfirmedChan() chan string {
	return make(chan string)
}

func (d *HeadlessDaemon) ConfigSavedChan() chan interface{} {
	ch := make(chan interface{})
	close(ch)
	return ch
}

func (d *HeadlessDaemon) GetCurrentConfig() map[string]interface{} {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "getCurrentConfig"))
	m, err := d.StateManager.TryLoad()
	if err != nil {
		warn.Log("event", "state.missing", "err", err)
	}
	return m
}

func (d *HeadlessDaemon) SetProgress(progress Progress) {
	d.UI.Output(fmt.Sprintf("%s: %s", progress.Type, progress.Detail))
}

func (d *HeadlessDaemon) ClearProgress() {
}
