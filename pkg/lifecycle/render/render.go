package render

import (
	"context"

	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

var (
	ProgressLoad    = daemon.StringProgress("render", "load")
	ProgressResolve = daemon.StringProgress("render", "resolve")
	ProgressBuild   = daemon.StringProgress("render", "build")
	ProgressBackup  = daemon.StringProgress("render", "backup")
	ProgressExecute = daemon.StringProgress("render", "execute")
	ProgressCommit  = daemon.StringProgress("render", "commit")
)

// A renderer takes a resolved spec, collects config values, and renders assets
type renderer struct {
	Logger         log.Logger
	ConfigResolver config.Resolver
	Planner        planner.Planner
	StateManager   state.Manager
	Fs             afero.Afero
	UI             cli.Ui
	Daemon         daemon.Daemon
	Now            func() time.Time
}

func NewRenderer(
	logger log.Logger,
	fs afero.Afero,
	ui cli.Ui,
	stateManager state.Manager,
	planner planner.Planner,
	resolver config.Resolver,
	daemon daemon.Daemon,
) lifecycle.Renderer {
	return &renderer{
		Logger:         logger,
		ConfigResolver: resolver,
		Planner:        planner,
		StateManager:   stateManager,
		Fs:             fs,
		UI:             ui,
		Now:            time.Now,
		Daemon:         daemon,
	}
}

// Execute renders the assets and config
func (r *renderer) Execute(ctx context.Context, release *api.Release, step *api.Render) error {
	defer r.Daemon.ClearProgress()

	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute")

	r.Daemon.SetProgress(ProgressLoad)
	previousState, err := r.StateManager.TryLoad()
	if err != nil {
		return err
	}

	r.Daemon.SetProgress(ProgressResolve)
	templateContext, err := r.ConfigResolver.ResolveConfig(ctx, release, previousState.CurrentConfig())
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	debug.Log("event", "render.plan")
	r.Daemon.SetProgress(ProgressBuild)
	pln, err := r.Planner.Build(release.Spec.Assets.V1, release.Spec.Config.V1, release.Metadata, templateContext)
	if err != nil {
		return errors.Wrap(err, "build plan")

	}

	debug.Log("event", "backup.start")
	r.Daemon.SetProgress(ProgressBackup)
	err = r.backupIfPresent(constants.InstallerPrefixPath)
	if err != nil {
		return errors.Wrapf(err, "backup existing install directory %s", constants.InstallerPrefixPath)
	}

	r.Daemon.SetProgress(ProgressExecute)
	r.Daemon.SetStepName(ctx, daemon.StepNameConfirm)
	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}

	stateTemplateContext := make(map[string]interface{})
	for _, configGroup := range release.Spec.Config.V1 {
		for _, configItem := range configGroup.Items {
			if valueNotOverridenByDefault(configItem, templateContext, previousState.CurrentConfig()) {
				stateTemplateContext[configItem.Name] = templateContext[configItem.Name]
			}
		}
	}

	// edge case: empty config section of app yaml,
	// persist data from previous state.json
	if len(release.Spec.Config.V1) == 0 {
		stateTemplateContext = templateContext
	}

	r.Daemon.SetProgress(ProgressCommit)
	if err := r.StateManager.SerializeConfig(release.Spec.Assets.V1, release.Metadata, stateTemplateContext); err != nil {
		return errors.Wrap(err, "serialize state")
	}

	return nil
}

func (r *renderer) backupIfPresent(basePath string) error {
	exists, err := r.Fs.Exists(basePath)
	if err != nil {
		return errors.Wrapf(err, "check file exists")
	}
	if !exists {
		return nil
	}

	backupDest := fmt.Sprintf("%s.bak", basePath)
	level.Info(r.Logger).Log("step.type", "render", "event", "unpackTarget.backup.remove", "src", basePath, "dest", backupDest)
	if err := r.Fs.RemoveAll(backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s: remove existing %s", basePath, backupDest, backupDest)
	}
	if err := r.Fs.Rename(basePath, backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s", basePath, backupDest)
	}

	return nil
}

func valueNotOverridenByDefault(item *libyaml.ConfigItem, templateContext map[string]interface{}, savedState map[string]interface{}) bool {
	_, inSavedState := savedState[item.Name] // all values in savedState are non-default values

	if templateContext[item.Name] == "" {
		if inSavedState && savedState[item.Name] == "" {
			// manually set value: "" in state.json
			return true
		} else {
			// value overriden by default == ""?
			return item.Default != ""
		}
	} else if templateContext[item.Name] == item.Default {
		// the provided value is manually set to the default value
		return inSavedState
	} else {
		// non-empty value != default. cannot have been overriden by default
		return true
	}
}
