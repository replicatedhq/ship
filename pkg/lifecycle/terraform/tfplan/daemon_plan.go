package tfplan

import (
	"context"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
)

type PlanConfirmer interface {
	ConfirmPlan(
		ctx context.Context,
		formmatedTerraformPlan string,
		step api.Terraform,
		release api.Release,
	) (bool, error)
}

func NewPlanner(
	logger log.Logger,
	daemon daemon.Daemon,
) PlanConfirmer {
	return &DaemonPlanner{
		Logger: logger,
		Daemon: daemon,
	}
}

// DaemonPlanner interfaces with the Daemon
// to perform interactions with the end user
type DaemonPlanner struct {
	Logger log.Logger
	Daemon daemon.Daemon
}

// ConfirmPlan presents the plan to the user.
// returns true if the plan should be applied
func (d *DaemonPlanner) ConfirmPlan(
	ctx context.Context,
	formmatedTerraformPlan string,
	step api.Terraform,
	release api.Release,
) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonplanner", "method", "plan"))

	debug.Log("event", "step.push")
	daemonExitedChan := d.Daemon.EnsureStarted(ctx, &release)
	d.Daemon.PushMessageStep(
		ctx,
		daemon.Message{Contents: formmatedTerraformPlan, TrustedHTML: true},
		planActions(),
		api.Step{Terraform: &step},
	)

	shouldApply, err := d.awaitPlanResult(ctx, daemonExitedChan)
	return shouldApply, errors.Wrap(err, "await plan confirm")

}

func (d *DaemonPlanner) awaitPlanResult(ctx context.Context, daemonExitedChan chan error) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonplanner", "method", "plan.confirm.await"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return false, ctx.Err()
		case err := <-daemonExitedChan:
			debug.Log("event", "daemon.exit")
			if err != nil {
				return false, err
			}
			return false, errors.New("daemon exited")
		case applyRequested := <-d.Daemon.TerraformConfirmedChan():
			debug.Log("event", "plan.confirmed", "applyRequested", applyRequested)
			return applyRequested, nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "plan.confirmed")
		}
	}
}

func planActions() []daemon.Action {
	return []daemon.Action{
		{
			ButtonType:  "primary",
			Text:        "Apply",
			LoadingText: "Applying",
			OnClick: daemon.ActionRequest{
				URI:    "/terraform/apply",
				Method: "POST",
			},
		},
		{
			ButtonType:  "secondary-gray",
			Text:        "Skip",
			LoadingText: "Skipping",
			OnClick: daemon.ActionRequest{
				URI:    "/terraform/skip",
				Method: "POST",
			},
		},
	}
}
