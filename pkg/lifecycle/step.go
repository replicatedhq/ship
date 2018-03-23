package lifecycle

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
)

var _ Executor = &stepExecutor{}

type stepExecutor struct {
	step *api.Step
}

func (s *stepExecutor) Execute(ctx context.Context, runner *Runner) error {
	debug := level.Debug(log.With(runner.Logger, "method", "execute"))

	if s.step.Message != nil {
		debug.Log("event", "step.resolve", "type", "message")
		err := (&messageExecutor{s.step.Message}).Execute(ctx, runner)
		debug.Log("event", "step.complete", "type", "message", "err", err)
		return errors.Wrap(err, "execute message step")
	} else if s.step.Render != nil {
		debug.Log("event", "step.resolve", "type", "render")
		err := (&render.Renderer{
			Step:   s.step.Render,
			Fs:     runner.Fs,
			Logger: runner.Logger,
			Spec:   runner.Spec,
			UI:     runner.UI,
			Viper:  runner.Viper,
			ConfigResolver: &render.ConfigResolver{
				Step:   s.step.Render,
				Fs:     runner.Fs,
				Logger: runner.Logger,
				Spec:   runner.Spec,
				UI:     runner.UI,
				Viper:  runner.Viper,
			},
		}).Execute(ctx)
		debug.Log("event", "step.complete", "type", "render", "err", err)
		return errors.Wrap(err, "execute render step")
	}

	return nil
}

func (s *stepExecutor) String() string {
	return fmt.Sprintf("Step{step=%v}", s.step)
}
