package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/message"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"go.uber.org/dig"
)

type StepExecutor struct {
	dig.In

	Logger    log.Logger
	Renderer  *render.Renderer
	Messenger message.Messenger
	Daemon    config.Daemon
}

func (s *StepExecutor) Execute(ctx context.Context, release *api.Release, step *api.Step) error {
	debug := level.Debug(log.With(s.Logger, "method", "execute"))

	if step.Message != nil {
		debug.Log("event", "step.resolve", "type", "message")
		err := s.Messenger.Execute(ctx, release, step.Message)
		debug.Log("event", "step.complete", "type", "message", "err", err)
		return errors.Wrap(err, "execute message step")
	} else if step.Render != nil {
		debug.Log("event", "step.resolve", "type", "render")
		err := s.Renderer.Execute(ctx, release, step.Render)
		debug.Log("event", "step.complete", "type", "render", "err", err)
		return errors.Wrap(err, "execute render step")
	}

	return nil
}

func (s *StepExecutor) End(ctx context.Context) error {
	s.Daemon.AllStepsDone(ctx)
	return nil
}
