package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
)

type HeadlessDaemon struct {
	StateManager   *state.Manager
	Logger         log.Logger
	UI             cli.Ui
	ConfigRenderer *APIConfigRenderer
	ResolvedConfig map[string]interface{}
}

func (d *HeadlessDaemon) EnsureStarted(ctx context.Context, release *api.Release) chan error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "EnsureStarted"))

	chanerrors := make(chan error)

	if err := d.HeadlessResolve(ctx, release); err != nil {
		warn.Log("event", "headless resolved failed", "err", err)
		go func() {
			chanerrors <- err
			close(chanerrors)
		}()
	}

	return chanerrors
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
	if d.ResolvedConfig != nil {
		return d.ResolvedConfig
	}

	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "getCurrentConfig"))
	currentConfig, err := d.StateManager.TryLoad()
	if err != nil {
		warn.Log("event", "state missing", "err", err)
	}

	return currentConfig
}

func (d *HeadlessDaemon) HeadlessResolve(ctx context.Context, release *api.Release) error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "HeadlessResolve"))
	currentConfig := d.GetCurrentConfig()

	resolved, err := d.ConfigRenderer.ResolveConfig(ctx, release, currentConfig, make(map[string]interface{}))
	if err != nil {
		warn.Log("event", "resolveconfig failed", "err", err)
		return err
	}

	if validateState := validateConfig(resolved); validateState != nil {
		var invalidItemNames []string
		for _, invalidConfigItems := range validateState {
			invalidItemNames = append(invalidItemNames, invalidConfigItems.Name)
		}

		err := errors.Errorf(
			"validate config failed. missing config values: %s",
			strings.Join(invalidItemNames, ","),
		)
		warn.Log("event", "state invalid", "err", err)
		return err
	}

	templateContext := make(map[string]interface{})
	for _, configGroup := range resolved {
		for _, configItem := range configGroup.Items {
			templateContext[configItem.Name] = configItem.Value
		}
	}

	d.ResolvedConfig = templateContext
	if err := d.StateManager.Serialize(nil, api.ReleaseMetadata{}, templateContext); err != nil {
		warn.Log("msg", "serialize state failed", "err", err)
		return err
	}

	return nil
}

func (d *HeadlessDaemon) SetProgress(progress Progress) {
	d.UI.Output(fmt.Sprintf("%s: %s", progress.Type, progress.Detail))
}

func (d *HeadlessDaemon) ClearProgress() {}
