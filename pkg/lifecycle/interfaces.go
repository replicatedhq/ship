package lifecycle

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
)

type Messenger interface {
	Execute(ctx context.Context, release *api.Release, step *api.Message) error
}

type Renderer interface {
	Execute(ctx context.Context, release *api.Release, step *api.Render) error
}

type Terraformer interface {
	Execute(ctx context.Context, release api.Release, step api.Terraform) error
}

type HelmIntro interface {
	Execute(context.Context, *api.Release, *api.HelmIntro) error
}

type HelmValues interface {
	Execute(context.Context, *api.Release, *api.HelmValues) error
}

type Kustomizer interface {
	Execute(ctx context.Context, release api.Release, step api.Kustomize) error
}

type Kubectl interface {
	Execute(ctx context.Context, release api.Release, step api.Kubectl) error
}
