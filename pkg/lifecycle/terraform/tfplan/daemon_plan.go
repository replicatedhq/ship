package tfplan

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

type PlanConfirmer interface {
	ConfirmPlan(
		ctx context.Context,
		formmatedTerraformPlan string,
		release api.Release,
		confirmedChan chan bool,
	) (bool, error)
	WithStatusReceiver(daemontypes.StatusReceiver) PlanConfirmer
}

func NewPlanner(
	logger log.Logger,
	daemon daemontypes.Daemon,
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
	Daemon daemontypes.Daemon
}

// WithStatusReceiver is a no-op for the daemon version of PlanConfirmer
func (d *DaemonPlanner) WithStatusReceiver(daemontypes.StatusReceiver) PlanConfirmer {
	return d
}

// ConfirmPlan presents the plan to the user.
// returns true if the plan should be applied
func (d *DaemonPlanner) ConfirmPlan(
	ctx context.Context,
	formmatedTerraformPlan string,
	release api.Release,
	confirmedChan chan bool,
) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonplanner", "method", "plan"))

	debug.Log("event", "step.push")
	daemonExitedChan := d.Daemon.EnsureStarted(ctx, &release)
	d.Daemon.PushMessageStep(
		ctx,
		daemontypes.Message{Contents: formmatedTerraformPlan, TrustedHTML: true},
		planActions(),
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

func planActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			ButtonType:  "primary",
			Text:        "Apply",
			LoadingText: "Applying",
			OnClick: daemontypes.ActionRequest{
				URI:    "/terraform/apply",
				Method: "POST",
			},
		},
		{
			ButtonType:  "secondary-gray",
			Text:        "Skip",
			LoadingText: "Skipping",
			OnClick: daemontypes.ActionRequest{
				URI:    "/terraform/skip",
				Method: "POST",
			},
		},
	}
}
