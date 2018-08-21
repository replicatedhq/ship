package render

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"go.uber.org/dig"
)

func NoConfigRenderer(render noconfigrenderer) lifecycle.Renderer {
	return &render
}

// noconfigrenderer is the navcycle version of
// render, that assumes that config has already happened
type noconfigrenderer struct {
	dig.In
	Logger         log.Logger
	Planner        planner.Planner
	StateManager   state.Manager
	Fs             afero.Afero
	UI             cli.Ui
	StatusReceiver daemontypes.StatusReceiver
	Now            func() time.Time
}

func (r *noconfigrenderer) Execute(ctx context.Context, release *api.Release, step *api.Render) error {
	defer r.StatusReceiver.ClearProgress()

	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute")

	r.StatusReceiver.SetProgress(ProgressRead)
	debug.Log("event", "try.load")
	previousState, err := r.StateManager.TryLoad()
	if err != nil {
		return err
	}

	templateContext := previousState.CurrentConfig()
	r.StatusReceiver.SetProgress(ProgressRender)

	debug.Log("event", "render.plan")
	pln, err := r.Planner.Build(step.Root, release.Spec.Assets.V1, release.Spec.Config.V1, release.Metadata, templateContext)
	if err != nil {
		return errors.Wrap(err, "build plan")
	}

	debug.Log("event", "backup.start")

	if step.Root == "" {
		step.Root = constants.InstallerPrefixPath
	}

	if step.Root != "." {
		err = r.backupIfPresent(step.Root)
		if err != nil {
			return errors.Wrapf(err, "backup existing install directory %s", constants.InstallerPrefixPath)
		}
	}

	debug.Log("event", "execute.plan")
	err = r.Planner.Execute(ctx, pln)
	if err != nil {
		return errors.Wrap(err, "execute plan")
	}
	return nil
}

func (r *noconfigrenderer) WithStatusReceiver(receiver daemontypes.StatusReceiver) lifecycle.Renderer {
	return &noconfigrenderer{
		Logger:         r.Logger,
		Planner:        r.Planner,
		StateManager:   r.StateManager,
		Fs:             r.Fs,
		UI:             r.UI,
		StatusReceiver: receiver,
		Now:            r.Now,
	}
}

func (r *noconfigrenderer) WithPlanner(planner planner.Planner) lifecycle.Renderer {
	return &noconfigrenderer{
		Logger:         r.Logger,
		Planner:        planner,
		StateManager:   r.StateManager,
		Fs:             r.Fs,
		UI:             r.UI,
		StatusReceiver: r.StatusReceiver,
		Now:            r.Now,
	}
}
