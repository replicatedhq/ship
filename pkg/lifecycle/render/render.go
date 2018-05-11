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

// A Renderer takes a resolved spec, collects config values, and renders assets
type Renderer struct {
	Logger         log.Logger
	ConfigResolver config.Resolver
	Planner        planner.Planner
	StateManager   *state.StateManager
	Fs             afero.Afero
	UI             cli.Ui
	Daemon         *config.Daemon
}

func FromViper(v *viper.Viper) *Renderer {
	return &Renderer{
		Logger:         logger.FromViper(v),
		ConfigResolver: config.ResolverFromViper(v),
		Planner:        planner.FromViper(v),
		StateManager:   state.ManagerFromViper(v),
		Fs:             fs.FromViper(v),
		UI:             ui.FromViper(v),
	}
}

func (r *Renderer) WithDaemon(d *config.Daemon) *Renderer {
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

	r.Daemon.SetProgress(config.StringProgress("render", "load"))
	previousTemplateContext, err := r.StateManager.TryLoad()
	if err != nil {
		return err
	}

	r.Daemon.SetProgress(config.StringProgress("render", "resolve"))
	templateContext, err := r.ConfigResolver.ResolveConfig(ctx, release, previousTemplateContext)
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	debug.Log("event", "render.plan")
	r.Daemon.SetProgress(config.StringProgress("render", "build"))
	pln := r.Planner.Build(release.Spec.Assets.V1, release.Spec.Config.V1, release.Metadata, templateContext)

	debug.Log("event", "render.plan.skip")

	r.Daemon.SetProgress(config.StringProgress("render", "execute"))
	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}

	r.Daemon.SetProgress(config.StringProgress("render", "commit"))
	if err := r.StateManager.Serialize(release.Spec.Assets.V1, release.Metadata, templateContext); err != nil {
		return err
	}

	return nil
}
