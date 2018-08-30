package tfplan

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

type DaemonlessPlanner struct {
	Logger log.Logger
	Status daemontypes.StatusReceiver
}

func NewDaemonlessPlanner(
	logger log.Logger,
) PlanConfirmer {
	return &DaemonlessPlanner{
		Logger: logger,
	}
}

func (d *DaemonlessPlanner) WithStatusReceiver(statusReceiver daemontypes.StatusReceiver) PlanConfirmer {
	return &DaemonlessPlanner{
		Logger: d.Logger,
		Status: statusReceiver,
	}
}

func (d *DaemonlessPlanner) ConfirmPlan(
	ctx context.Context,
	formattedTerraformPlan string,
	release api.Release,
	confirmedChan chan bool,
) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonlessplanner", "method", "plan"))

	debug.Log("event", "status.message")
	d.Status.PushMessageStep(
		ctx,
		daemontypes.Message{Contents: formattedTerraformPlan, TrustedHTML: true},
		planActions(),
	)

	return d.awaitMessageConfirmed(ctx, confirmedChan)
}

func (d *DaemonlessPlanner) awaitMessageConfirmed(ctx context.Context, confirmedChan chan bool) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonlessplanner", "method", "awaitMessageConfirmed"))

	for {
		select {
		case shouldApply := <-confirmedChan:
			return shouldApply, nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}
