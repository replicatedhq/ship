package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/helmIntro"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/lifecycle/message"
	"github.com/replicatedhq/ship/pkg/lifecycle/render"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform"
	"go.uber.org/dig"
)

type StepExecutor struct {
	dig.In

	Logger      log.Logger
	Renderer    *render.Renderer
	Messenger   message.Messenger
	Terraformer terraform.Terraformer
	Daemon      daemon.Daemon
	Kustomizer  kustomize.Kustomizer
	HelmIntro   helmIntro.HelmIntro
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
	} else if step.Terraform != nil {
		debug.Log("event", "step.resolve", "type", "terraform")
		err := s.Terraformer.Execute(ctx, *release, *step.Terraform)
		debug.Log("event", "step.complete", "type", "terraform", "err", err)
		return errors.Wrap(err, "execute terraform step")
	} else if step.Kustomize != nil {
		debug.Log("event", "step.resolve", "type", "kustomize")
		err := s.Kustomizer.Execute(ctx, *release, *step.Kustomize)
		debug.Log("event", "step.complete", "type", "kustomize", "err", err)
		return errors.Wrap(err, "execute kustomize step")
	} else if step.HelmIntro != nil {
		debug.Log("event", "step.helmIntro", "type", "helmIntro")
		err := s.HelmIntro.Execute(ctx, release, step.HelmIntro)
		debug.Log("event", "step.complete", "type", "helmIntro", "err", err)
	}

	debug.Log("event", "step.unknown")
	return nil
}

func (s *StepExecutor) End(ctx context.Context) error {
	s.Daemon.AllStepsDone(ctx)
	return nil
}
