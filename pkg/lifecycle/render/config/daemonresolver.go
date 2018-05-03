package config

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
)

const StepNameConfig = "render.config"
const StepNameConfirm = "render.confirm"

type DaemonResolver struct {
	Logger             log.Logger
	MaybeRunningDaemon *Daemon
}

func (d *DaemonResolver) ResolveConfig(
	ctx context.Context,
	release *api.Release,
	context map[string]interface{},
) (map[string]interface{}, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonresolver", "method", "resolveConfig"))
	if len(release.Spec.Config.V1) == 0 {
		debug.Log("event", "config.empty")
		return context, nil
	}

	daemonExitedChan := d.MaybeRunningDaemon.EnsureStarted(ctx, release)

	for _, step := range release.Spec.Lifecycle.V1 {
		if step.Render != nil {
			debug.Log("event", "render.found")
			d.MaybeRunningDaemon.PushStep(ctx, StepNameConfig, step)
			debug.Log("event", "step.pushed")
			select {
			case <-ctx.Done():
				debug.Log("event", "ctx.done")
				return nil, ctx.Err()
			case err := <-daemonExitedChan:
				debug.Log("event", "daemon.exit")
				return nil, err
			case <-d.MaybeRunningDaemon.ConfigSaved:
				debug.Log("event", "config.saved")
				return d.MaybeRunningDaemon.CurrentConfig, nil
			}
		}
	}

	return nil, errors.New("couldn't find current render Step")
}
