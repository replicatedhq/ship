package render

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	pkgplanner "github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

var (
	ProgressRead   = daemontypes.StringProgress("render", "reading application release")
	ProgressRender = daemontypes.StringProgress("render", "rendering assets and configuration values")
)

// A headlessrenderer takes a resolved spec, collects config values, and renders assets
// this was the old plain "renderer", but now its only used in headless workflows, so renaming it for clarity
type headlessrenderer struct {
	Logger         log.Logger
	ConfigResolver config.Resolver
	Planner        pkgplanner.Planner
	StateManager   state.Manager
	Fs             afero.Afero
	UI             cli.Ui
	StatusReceiver daemontypes.StatusReceiver
	Now            func() time.Time
}

// Execute renders the assets and config
func (r *headlessrenderer) Execute(ctx context.Context, release *api.Release, step *api.Render) error {
	defer r.StatusReceiver.ClearProgress()

	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute")

	err := r.StateManager.ClearListsMetadata()
	if err != nil {
		return errors.Wrap(err, "clear list metadata")
	}

	debug.Log("event", "try.load")
	previousState, err := r.StateManager.TryLoad()
	if err != nil {
		return err
	}

	r.StatusReceiver.SetProgress(ProgressRead)

	debug.Log("event", "resolve.config")
	templateContext, err := r.ConfigResolver.ResolveConfig(ctx, release, previousState.CurrentConfig())
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	r.StatusReceiver.SetProgress(ProgressRender)

	// this should probably happen even higher up, like to the validation stage where we assign IDs to lifecycle steps, but moving it up here for now
	if step.Root == "" {
		step.Root = constants.InstallerPrefixPath
	}

	assets := release.Spec.Assets.V1
	if step.Assets != nil && step.Assets.V1 != nil {
		assets = step.Assets.V1
	}

	debug.Log("event", "render.plan")
	pln, err := r.Planner.Build(step.Root, assets, release.Spec.Config.V1, release.Metadata, templateContext)
	if err != nil {
		return errors.Wrap(err, "build plan")
	}

	if step.Root != "." && step.Root != "./" {
		debug.Log("event", "backup.start")
		err = r.backupIfPresent(step.Root)
		if err != nil {
			return errors.Wrapf(err, "backup existing install directory %s", constants.InstallerPrefixPath)
		}
	}

	debug.Log("event", "execute.plan")
	r.StatusReceiver.SetStepName(ctx, daemontypes.StepNameConfirm)
	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}

	stateTemplateContext := make(map[string]interface{})
	for _, configGroup := range release.Spec.Config.V1 {
		for _, configItem := range configGroup.Items {
			if valueNotOverriddenByDefault(configItem, templateContext, previousState.CurrentConfig()) {
				stateTemplateContext[configItem.Name] = templateContext[configItem.Name]
			}
		}
	}

	// edge case: empty config section of app yaml,
	// persist data from previous state.json
	if len(release.Spec.Config.V1) == 0 {
		stateTemplateContext = templateContext
	}

	debug.Log("event", "commit")
	if err := r.StateManager.SerializeConfig(release.Spec.Assets.V1, release.Metadata, stateTemplateContext); err != nil {
		return errors.Wrap(err, "serialize state")
	}

	return nil
}

func valueNotOverriddenByDefault(item *libyaml.ConfigItem, templateContext map[string]interface{}, savedState map[string]interface{}) bool {
	_, inSavedState := savedState[item.Name] // all values in savedState are non-default values

	if templateContext[item.Name] == "" {
		if inSavedState && savedState[item.Name] == "" {
			// manually set value: "" in state.json
			return true
		} else {
			// value overridden by default == ""?
			return item.Default != ""
		}
	} else if templateContext[item.Name] == item.Default {
		// the provided value is manually set to the default value
		return inSavedState
	} else {
		// non-empty value != default. cannot have been overridden by default
		return true
	}
}
