package kustomize

import (
	"context"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
)

type Kustomizer interface {
	Execute(ctx context.Context, release api.Release, step api.Kustomize) error
}

func NewKustomizer(
	logger log.Logger,
	daemon daemon.Daemon,
) Kustomizer {
	return &kustomizer{
		Logger: logger,
		Daemon: daemon,
	}

}

// kustomizer will *try* to pull in the Kustomizer libs from kubernetes-sigs/kustomize,
// if not we'll have to fork. for now it just explodes
type kustomizer struct {
	Logger log.Logger
	Daemon daemon.Daemon
}

func (l *kustomizer) Execute(ctx context.Context, release api.Release, step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "method", "execute"))

	daemonExitedChan := l.Daemon.EnsureStarted(ctx, &release)

	debug.Log("event", "daemon.started")

	l.Daemon.PushKustomizeStep(ctx)
	debug.Log("event", "step.pushed")

	return l.awaitMessageConfirmed(ctx, daemonExitedChan)
}

func (l *kustomizer) awaitMessageConfirmed(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "method", "kustomize.save.await"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return ctx.Err()
		case err := <-daemonExitedChan:
			debug.Log("event", "daemon.exit")
			if err != nil {
				return err
			}
			return errors.New("daemon exited")
		case <-l.Daemon.KustomizeSavedChan():
			debug.Log("event", "kustomize.finalized")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "kustomize.finalized")
		}
	}
}
