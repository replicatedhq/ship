package tfplan

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
)

type DaemonlessPlanner struct {
	Logger log.Logger
}

func NewDaemonlessPlanner(
	logger log.Logger,
) PlanConfirmer {
	return &DaemonlessPlanner{
		Logger: logger,
	}
}

func (d *DaemonlessPlanner) ConfirmPlan(
	ctx context.Context,
	formmatedTerraformPlan string,
	release api.Release,
) (bool, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonlessplanner", "method", "plan"))

	debug.Log("no-op")
	return true, nil
}
