package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
)

type stepExecutor struct {
	Logger    log.Logger
	renderer  *render.Renderer
	messenger *messenger
}

func (s *stepExecutor) Execute(ctx context.Context, step *api.Step) error {
	debug := level.Debug(log.With(s.Logger, "method", "execute"))

	if step.Message != nil {
		debug.Log("event", "step.resolve", "type", "message")
		err := s.messenger.Execute(ctx, step.Message)
		debug.Log("event", "step.complete", "type", "message", "err", err)
		return errors.Wrap(err, "execute message step")
	} else if step.Render != nil {
		debug.Log("event", "step.resolve", "type", "render")
		err := s.renderer.Execute(ctx, step.Render)
		debug.Log("event", "step.complete", "type", "render", "err", err)
		return errors.Wrap(err, "execute render step")
	}

	return nil
}
