package render

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	pkgplanner "github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

// Factory gets a *renderer and implements lifecycle.Renderer
type Factory func() *renderer

// factory implements lifecycle.Renderer
var _ lifecycle.Renderer = Factory(func() *renderer { return nil })

func (f Factory) Execute(ctx context.Context, release *api.Release, step *api.Render) error {
	r := f()
	return r.Execute(ctx, release, step)
}

func (f Factory) WithPlanner(plannerFactory pkgplanner.Planner) lifecycle.Renderer {
	return Factory(func() *renderer {
		r := f()
		return &renderer{
			Logger:         r.Logger,
			ConfigResolver: r.ConfigResolver,
			Planner:        plannerFactory,
			StateManager:   r.StateManager,
			Fs:             r.Fs,
			UI:             r.UI,
			Now:            time.Now,
			StatusReceiver: r.StatusReceiver,
		}
	})
}

func (f Factory) WithStatusReceiver(receiver daemontypes.StatusReceiver) lifecycle.Renderer {
	return Factory(func() *renderer {
		r := f()
		return &renderer{
			Logger:         r.Logger,
			ConfigResolver: r.ConfigResolver,
			Planner:        r.Planner,
			StateManager:   r.StateManager,
			Fs:             r.Fs,
			UI:             r.UI,
			Now:            time.Now,
			StatusReceiver: receiver,
		}
	})
}

func NewFactory(
	logger log.Logger,
	fs afero.Afero,
	ui cli.Ui,
	stateManager state.Manager,
	planner pkgplanner.Planner,
	resolver config.Resolver,
	status daemontypes.StatusReceiver,
) lifecycle.Renderer {
	return Factory(func() *renderer {
		return &renderer{
			Logger:         logger,
			ConfigResolver: resolver,
			Planner:        planner,
			StateManager:   stateManager,
			Fs:             fs,
			UI:             ui,
			Now:            time.Now,
			StatusReceiver: status,
		}
	})
}
