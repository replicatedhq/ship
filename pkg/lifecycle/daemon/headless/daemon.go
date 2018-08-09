package headless

import (
	"context"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/state"
)

var _ daemontypes.Daemon = &HeadlessDaemon{}

type HeadlessDaemon struct {
	StateManager   state.Manager
	Logger         log.Logger
	UI             cli.Ui
	ConfigRenderer *resolve.APIConfigRenderer
	ResolvedConfig map[string]interface{}
}

func (d *HeadlessDaemon) AwaitShutdown() error {
	return nil
}

func NewHeadlessDaemon(
	ui cli.Ui,
	logger log.Logger,
	renderer *resolve.APIConfigRenderer,
	stateManager state.Manager,
) daemontypes.Daemon {
	return &HeadlessDaemon{
		StateManager:   stateManager,
		Logger:         logger,
		UI:             ui,
		ConfigRenderer: renderer,
	}
}

func (d *HeadlessDaemon) PushKustomizeStep(context.Context, daemontypes.Kustomize)                   {}
func (d *HeadlessDaemon) PushMessageStep(context.Context, daemontypes.Message, []daemontypes.Action) {}
func (d *HeadlessDaemon) PushRenderStep(context.Context, daemontypes.Render)                         {}

func (d *HeadlessDaemon) KustomizeSavedChan() chan interface{} {
	ch := make(chan interface{}, 1)
	level.Debug(d.Logger).Log("event", "kustomize.skip", "detail", "running in automation, not waiting for kustomize")
	ch <- nil
	return ch
}

func (d *HeadlessDaemon) PushHelmIntroStep(context.Context, daemontypes.HelmIntro, []daemontypes.Action) {
}

func (d *HeadlessDaemon) PushHelmValuesStep(ctx context.Context, helmValues daemontypes.HelmValues, actions []daemontypes.Action) {
	warn := level.Warn(log.With(d.Logger, "struct", "HeadlessDaemon", "method", "PushHelmValuesStep"))
	if err := d.HeadlessSaveHelmValues(ctx, helmValues.Values); err != nil {
		warn.Log("event", "push helm values step fail", "err", err)
	}
}

func (d *HeadlessDaemon) HeadlessSaveHelmValues(ctx context.Context, helmValues string) error {
	warn := level.Warn(log.With(d.Logger, "struct", "HeadlessDaemon", "method", "HeadlessSaveHelmValues"))
	err := d.StateManager.SerializeHelmValues(helmValues)
	if err != nil {
		warn.Log("event", "headless save helm values fail", "err", err)
		return err
	}

	return nil
}

func (d *HeadlessDaemon) PushStreamStep(context.Context, <-chan daemontypes.Message) {}

func (d *HeadlessDaemon) CleanPreviousStep() {}

// todo I think if headless we should blow up here, but for now just skipping
func (d *HeadlessDaemon) TerraformConfirmedChan() chan bool {
	ch := make(chan bool, 1)
	level.Debug(d.Logger).Log("event", "terraform.skip", "detail", "running in automation, auto-skipping terraform plan")
	ch <- false
	return ch
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

func (d *HeadlessDaemon) SetStepName(context.Context, string) {}

func (d *HeadlessDaemon) AllStepsDone(context.Context) {}

func (d *HeadlessDaemon) MessageConfirmedChan() chan string {
	ch := make(chan string)
	close(ch)
	return ch
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

	return currentConfig.CurrentConfig()
}

func (d *HeadlessDaemon) HeadlessResolve(ctx context.Context, release *api.Release) error {
	warn := level.Warn(log.With(d.Logger, "struct", "fakeDaemon", "method", "HeadlessResolve"))
	currentConfig := d.GetCurrentConfig()

	resolved, err := d.ConfigRenderer.ResolveConfig(ctx, release, currentConfig, make(map[string]interface{}), false)
	if err != nil {
		warn.Log("event", "resolveconfig failed", "err", err)
		return err
	}

	if validateState := resolve.ValidateConfig(resolved); validateState != nil {
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
	if err := d.StateManager.SerializeConfig(nil, api.ReleaseMetadata{}, templateContext); err != nil {
		warn.Log("msg", "serialize state failed", "err", err)
		return err
	}

	return nil
}

func (d *HeadlessDaemon) SetProgress(progress daemontypes.Progress) {
	d.UI.Output(progress.Detail)
}

func (d *HeadlessDaemon) ClearProgress() {}
