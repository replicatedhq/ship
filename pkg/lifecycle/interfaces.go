package lifecycle

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
)

type DoWithProgress func(progress daemontypes.Progress)
type DoWithMessage func(progress daemontypes.Message)

type Messenger interface {
	Execute(ctx context.Context, release *api.Release, step *api.Message) error
}

type Renderer interface {
	Execute(ctx context.Context, release *api.Release, step *api.Render) error
	WithStatusReceiver(receiver daemontypes.StatusReceiver) Renderer
	WithPlanner(planner planner.Planner) Renderer
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

type KubectlApply interface {
	Execute(ctx context.Context, release api.Release, step api.KubectlApply) error
}
