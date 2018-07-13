package terraform

import (
	"context"

	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
)

// DaemonPlanner interfaces with the Daemon
// to perform interactions with the end user
type DaemonPlanner struct {
	Daemon daemon.Daemon
}

func (d *DaemonPlanner) Plan(
	ctx context.Context,
	formmatedTerraformPlan string,
) {
	d.Daemon.PushMessageStep(
		ctx,
		daemon.Message{Contents: formmatedTerraformPlan},
		daemon.TerraformActions(),
	)
}
