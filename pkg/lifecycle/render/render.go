package render

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/fs"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// StateFilePath is a placeholder for the default spot we'll store state. todo this should be a param or something
const StateFilePath = ".ship/state.json"

var (
	ProgressLoad    = config.StringProgress("render", "load")
	ProgressResolve = config.StringProgress("render", "resolve")
	ProgressBuild   = config.StringProgress("render", "build")
	ProgressExecute = config.StringProgress("render", "execute")
	ProgressCommit  = config.StringProgress("render", "commit")
)

// A Renderer takes a resolved spec, collects config values, and renders assets
type Renderer struct {
	Logger         log.Logger
	ConfigResolver config.Resolver
	Planner        planner.Planner
	StateManager   *state.StateManager
	Fs             afero.Afero
	UI             cli.Ui
	Daemon         config.Daemon
}

func FromViper(v *viper.Viper) (*Renderer, error) {

	pln, err := planner.FromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "initialize planner")
	}

	return &Renderer{
		Logger:         logger.FromViper(v),
		ConfigResolver: config.ResolverFromViper(v),
		Planner:        pln,
		StateManager:   state.ManagerFromViper(v),
		Fs:             fs.FromViper(v),
		UI:             ui.FromViper(v),
	}, nil
}

func (r *Renderer) WithDaemon(d config.Daemon) *Renderer {
	r.Daemon = d
	r.ConfigResolver = r.ConfigResolver.WithDaemon(d)
	r.Planner = r.Planner.WithDaemon(d)
	return r
}

// Execute renders the assets and config
func (r *Renderer) Execute(ctx context.Context, release *api.Release, step *api.Render) error {
	defer r.Daemon.ClearProgress()

	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute", "step.skipPlan", step.SkipPlan)

	r.Daemon.SetProgress(ProgressLoad)
	previousTemplateContext, err := r.StateManager.TryLoad()
	if err != nil {
		return err
	}

	r.Daemon.SetProgress(ProgressResolve)
	templateContext, err := r.ConfigResolver.ResolveConfig(ctx, release, previousTemplateContext)
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	debug.Log("event", "render.plan")
	r.Daemon.SetProgress(ProgressBuild)
	pln := r.Planner.Build(release.Spec.Assets.V1, release.Spec.Config.V1, release.Metadata, templateContext)

	debug.Log("event", "render.plan.skip")

	r.Daemon.SetProgress(ProgressExecute)
	r.Daemon.SetStepName(ctx, config.StepNameConfirm)
	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}

	r.Daemon.SetProgress(ProgressCommit)
	if err := r.StateManager.Serialize(release.Spec.Assets.V1, release.Metadata, templateContext); err != nil {
		return err
	}

	return nil
}
