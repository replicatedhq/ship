package terraform

import (
	"context"

	"path"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/replicatedhq/ship/pkg/version"
)

type Terraformer interface {
	Execute(ctx context.Context, release api.Release, step api.Terraform) error
	WithDaemon(d daemon.Daemon) Terraformer
}

type ForkTerraformer struct {
	Logger        log.Logger
	Daemon        daemon.Daemon
	PlanConfirmer tfplan.PlanConfirmer
}

func NewTerraformer(
	logger log.Logger,
	daemon daemon.Daemon,
	planner tfplan.PlanConfirmer,
) Terraformer {
	return &ForkTerraformer{
		Logger:        logger,
		Daemon:        daemon,
		PlanConfirmer: planner,
	}
}

func (t *ForkTerraformer) WithDaemon(daemon daemon.Daemon) Terraformer {
	return &ForkTerraformer{
		Logger: t.Logger,
		Daemon: daemon,
	}
}

func (t *ForkTerraformer) Execute(ctx context.Context, release api.Release, step api.Terraform) error {

	assetsPath := path.Join("/tmp", "ship-terraform", version.RunAtEpoch, "asset")
	_, err := t.plan(assetsPath)
	if err != nil {
		return errors.Wrapf(err, "create plan for %s", assetsPath)
	}
	// create plan, save to state
	// push infra plan step
	// maybe exit
	// set progress applying

	fakePlan := "We're gonna make you some servers"
	shouldApply, err := t.PlanConfirmer.ConfirmPlan(ctx, fakePlan, release)
	if err != nil {
		return errors.Wrapf(err, "confirm plan for %s", assetsPath)
	}

	if !shouldApply {
		return nil
	}

	// next: apply plan
	return nil
}

func (t *ForkTerraformer) plan(modulePath string) (string, error) {
	// we really shouldn't write plan to a file, but this will do for now
	planOut := path.Join("tmp", "ship-terraform", version.RunAtEpoch, "plan")
	return planOut, nil
}
