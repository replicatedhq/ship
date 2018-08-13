package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"go.uber.org/dig"
)

type StepExecutor struct {
	dig.In

	Logger       log.Logger
	Messenger    Messenger
	Renderer     Renderer
	Terraformer  Terraformer
	HelmIntro    HelmIntro
	HelmValues   HelmValues
	KubectlApply KubectlApply
	Kustomizer   Kustomizer
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
	} else if step.HelmIntro != nil && release.Metadata.HelmChartMetadata.Readme != "" {
		debug.Log("event", "step.helmIntro", "type", "helmIntro")
		err := s.HelmIntro.Execute(ctx, release, step.HelmIntro)
		debug.Log("event", "step.complete", "type", "helmIntro", "err", err)
	} else if step.HelmValues != nil {
		debug.Log("event", "step.helmValues", "type", "helmValues")
		err := s.HelmValues.Execute(ctx, release, step.HelmValues)
		debug.Log("event", "step.complete", "type", "helmValues", "err", err)
	} else if step.KubectlApply != nil {
		debug.Log("event", "step.resolve", "type", "kubectl")
		err := s.KubectlApply.Execute(ctx, *release, *step.KubectlApply)
		debug.Log("event", "step.complete", "type", "kubectl", "err", err)
	}

	debug.Log("event", "step.unknown", "name", step.ShortName(), "id", step.Shared().ID)
	return nil
}
