package lifecycle

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
)

var _ Executor = &renderExecutor{}

type renderExecutor struct {
	step *api.Render
}

func (r *renderExecutor) Execute(ctx context.Context, runner *Runner) error {
	debug := level.Debug(log.With(runner.Logger, "step.type", "render"))
	debug.Log("event", "step.execute", "step.plan", r.step.SkipPlan)
	// read runner.spec.config
	// gather config values
	// store to temp state
	// confirm? (obfuscating passwords)

	// read runner.spec.assets
	// build plan
	if r.step.SkipPlan {
		// print plan
		// confirm plan
	}
	// generate assets
	// save state
	return nil

	// on failure,
}

func (r *renderExecutor) String() string {
	return fmt.Sprintf("Render{SkipPlan=%v}", r.step.SkipPlan)
}
