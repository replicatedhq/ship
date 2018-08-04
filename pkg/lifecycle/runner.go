package lifecycle

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
)

// A Runner runs a lifecycle using the passed Spec
type Runner struct {
	Logger   log.Logger
	Executor *StepExecutor
}

func NewRunner(
	logger log.Logger,
	executor StepExecutor,
) *Runner {
	return &Runner{
		Logger:   logger,
		Executor: &executor,
	}
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
	return nil
}
