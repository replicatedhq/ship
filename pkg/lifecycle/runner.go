package lifecycle

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
)

// A Runner runs a lifecycle using the passed Spec
type Runner struct {
	Logger   log.Logger
	Executor *StepExecutor
}

/*

	// this needs to be pulled up more, but this is enough for now
	executor := &lifecycle.StepExecutor{
		Logger: s.Logger,
		Renderer: &render.Renderer{
			Fs:      s.Fs,
			Logger:  s.Logger,
			Release: s.Release,
			UI:      s.UI,
			ConfigResolver: &config.CLIResolver{
				Logger:  s.Logger,
				Release: s.Release,
				UI:      s.UI,
				Viper:   s.Viper,
			},
			Planner: &plan.CLIPlanner{
				Logger: s.Logger,
				Fs:     s.Fs,
				UI:     s.UI,
			},
			StateManager: &state.StateManager{
				Logger: s.Logger,
			},
		},
		messenger: &message.CLIMessenger{
			Logger: s.Logger,
			UI:     s.UI,
			Viper:  s.Viper,
		},
	}
*/

func NewRunner(logger log.Logger, executor StepExecutor) *Runner {
	return &Runner{
		Logger:   logger,
		Executor: &executor,
	}
}

func (r *Runner) WithDaemon(d config.Daemon) *Runner {
	r.Executor = r.Executor.WithDaemon(d)
	return r
}

func (e *StepExecutor) WithDaemon(d config.Daemon) *StepExecutor {
	e.Daemon = d
	e.Renderer = e.Renderer.WithDaemon(d)
	e.Messenger = e.Messenger.WithDaemon(d)
	return e
}

// Run runs a lifecycle using the passed Spec
func (r *Runner) Run(ctx context.Context, release *api.Release) error {
	level.Debug(r.Logger).Log("event", "lifecycle.execute")

	for idx, step := range release.Spec.Lifecycle.V1 {
		level.Debug(r.Logger).Log("event", "step.execute", "index", idx, "step", fmt.Sprintf("%v", step))
		if err := r.Executor.Execute(ctx, release, &step); err != nil {
			level.Error(r.Logger).Log("event", "step.execute.fail", "index", idx, "step", fmt.Sprintf("%v", step))
			return errors.Wrapf(err, "execute lifecycle step %d", idx)
		}
	}

	return r.Executor.End(ctx)
}
