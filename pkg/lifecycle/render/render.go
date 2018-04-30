package render

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/plan"
	"github.com/spf13/afero"
)

// StateFilePath is a placeholder for the default spot we'll store state. todo this should be a param or something
const StateFilePath = ".ship/state.json"

// A Renderer takes a resolved spec, collects config values, and renders assets
type Renderer struct {
	Logger         log.Logger
	ConfigResolver config.Resolver
	Planner        plan.Planner

	Fs      afero.Afero
	Release *api.Release
	UI      cli.Ui
}

// Execute renders the assets and config
func (r *Renderer) Execute(ctx context.Context, step *api.Render) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute", "step.skipPlan", step.SkipPlan)

	templateContext, err := r.ConfigResolver.ResolveConfig(&r.Release.Metadata, ctx)
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	debug.Log("event", "render.plan")
	pln := r.Planner.Build(r.Release.Spec.Assets.V1, r.Release.Metadata, templateContext)

	if !step.SkipPlan {
		debug.Log("event", "render.plan.confirm")
		planConfirmed, err := r.Planner.Confirm(pln)
		if err != nil {
			debug.Log("event", "render.plan.confirm.fail", "err", err)
			return errors.Wrap(err, "confirm plan")
		}
		if !planConfirmed {
			debug.Log("event", "render.plan.confirm.deny")
			return errors.New("plan denied")
		}
		debug.Log("event", "render.plan.confirm.confirm")
	} else {
		debug.Log("event", "render.plan.skip")
	}

	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}

	// if not studio:
	//      save state
	// else:
	//      warnStudio
	return nil
}
