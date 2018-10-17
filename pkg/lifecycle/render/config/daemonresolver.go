package config

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

type DaemonResolver struct {
	Logger log.Logger
	Daemon daemontypes.Daemon
}

func (d *DaemonResolver) ResolveConfig(
	ctx context.Context,
	release *api.Release,
	context map[string]interface{},
) (map[string]interface{}, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonresolver", "method", "resolveConfig"))
	if len(release.Spec.Config.V1) == 0 {
		debug.Log("event", "config.empty")
		if context == nil {
			return make(map[string]interface{}), nil
		}
		return context, nil
	}

	daemonExitedChan := d.Daemon.EnsureStarted(ctx, release)

	for _, step := range release.Spec.Lifecycle.V1 {
		if step.Render != nil {
			debug.Log("event", "render.found")
			d.Daemon.PushRenderStep(ctx, daemontypes.Render{})
			debug.Log("event", "step.pushed")
			return d.awaitConfigSaved(ctx, daemonExitedChan)
		}
	}

	return nil, errors.New("couldn't find current render Execute")
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
		case <-time.After(1 * time.Millisecond):
			// need to pause here to ensure err channel priority
		}
		select {
		case <-d.Daemon.ConfigSavedChan():
			debug.Log("event", "config.saved")
			return d.Daemon.GetCurrentConfig(), nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "config.saved")
		}
	}
}
