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
	"github.com/replicatedhq/libyaml"
)

type HeadlessDaemon struct {
	StateManager   *state.StateManager
	Logger         log.Logger
	UI             cli.Ui
	ConfigRenderer *APIConfigRenderer
}

func (d *HeadlessDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "EnsureStarted"))
	currentConfig := d.GetCurrentConfig()

	resolved, err := d.ConfigRenderer.ResolveConfig(ctx, release, currentConfig, currentConfig)
	if err != nil {
		warn.Log("event", "headless.resolved.failed", "err", err)
	}

	if err := d.ValidateSuppliedParams(resolved); err != nil {
		warn.Log("event", "headless.validate.failed", "err", err)
		d.UI.Error(err.Error())
		os.Exit(1)
	}

	templateContext := make(map[string]interface{})
	for _, configGroup := range resolved {
		for _, configItem := range configGroup.Items {
			templateContext[configItem.Name] = configItem.Value
		}
	}

	if err := d.StateManager.Serialize(nil, api.ReleaseMetadata{}, templateContext); err != nil {
		warn.Log("msg", "headless.serialize state failed", "err", err)
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
		warn.Log("event", "headless.state.missing", "err", err)
	}

	return currentConfig
}

func (d *HeadlessDaemon) ValidateSuppliedParams(resolved []libyaml.ConfigGroup) error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "validateSuppliedParams"))

	if validateState := validateConfig(resolved); validateState != nil {
		err := errors.New("Error: missing parameters. Exiting...")
		warn.Log("event", "headless.state.invalid", "err", err)
		return err
	}

	return nil
}

func (d *HeadlessDaemon) ChainConfig(currentConfig map[string]interface{}) map[string]interface{} {
	return nil
}

func (d *HeadlessDaemon) SetProgress(progress Progress) {
	d.UI.Output(fmt.Sprintf("%s: %s", progress.Type, progress.Detail))
}

func (d *HeadlessDaemon) ClearProgress() {}
