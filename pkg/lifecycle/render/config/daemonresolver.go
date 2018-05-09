package config

import (
	"context"
	"errors"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
)

const StepNameConfig = "render.config"
const StepNameConfirm = "render.confirm"

type DaemonResolver struct {
	Logger log.Logger
	Daemon *Daemon
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

	daemonExitedChan := d.Daemon.EnsureStarted(ctx, release)

	for _, step := range release.Spec.Lifecycle.V1 {
		if step.Render != nil {
			debug.Log("event", "render.found")
			d.Daemon.PushStep(ctx, StepNameConfig, step)
			debug.Log("event", "step.pushed")
			return d.awaitConfigSaved(ctx, daemonExitedChan)
		}
	}

	return nil, errors.New("couldn't find current render Step")
}

func (d *DaemonResolver) awaitConfigSaved(ctx context.Context, daemonExitedChan chan error) (map[string]interface{}, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonresolver", "method", "resolveConfig"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return nil, ctx.Err()
		case err := <-daemonExitedChan:
			debug.Log("event", "daemon.exit")
			if err != nil {
				return nil, err
			}
			return nil, errors.New("daemon exited")
		case <-d.Daemon.ConfigSaved:
			debug.Log("event", "config.saved")
			return d.Daemon.CurrentConfig, nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "config.saved")
		}
	}
}
