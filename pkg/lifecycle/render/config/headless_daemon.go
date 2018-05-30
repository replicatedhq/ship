package config

import (
	"context"
	"fmt"

	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
)

type HeadlessDaemon struct {
	StateManager   *state.StateManager
	Logger         log.Logger
	UI             cli.Ui
	ConfigRenderer *APIConfigRenderer
}

func (d *HeadlessDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "EnsureStarted"))
	if err := d.ValidateSuppliedParams(ctx, release); err != nil {
		warn.Log("event", "validate.failed", "err", err)
		d.UI.Error(err.Error())
		os.Exit(1)
	}
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
	currentConfig, err := d.StateManager.TryLoad()
	if err != nil {
		warn.Log("event", "state.missing", "err", err)
	}

	return currentConfig
}

func (d *HeadlessDaemon) ValidateSuppliedParams(ctx context.Context, release *api.Release) error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "validateSuppliedParams"))
	currentConfig, err := d.StateManager.TryLoad()
	if err != nil {
		warn.Log("event", "state.missing", "err", err)
		return err
	}

	resolved, err := d.ConfigRenderer.ResolveConfig(ctx, release, currentConfig, currentConfig)
	if err != nil {
		warn.Log("event", "resolve.failed", "err", err)
		return err
	}

	if validateState := validateConfig(resolved); validateState != nil {
		err := errors.New("Error: missing parameters. Exiting...")
		warn.Log("event", "state.invalid", "err", err)
		return err
	}

	return nil
}

func (d *HeadlessDaemon) SetProgress(progress Progress) {
	d.UI.Output(fmt.Sprintf("%s: %s", progress.Type, progress.Detail))
}

func (d *HeadlessDaemon) ClearProgress() {}
