package kustomize

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type Kustomizer struct {
	Logger log.Logger
	FS     afero.Afero
	State  state.Manager
}

func NewDaemonlessKustomizer(
	logger log.Logger,
	fs afero.Afero,
	state state.Manager,
) lifecycle.Kustomizer {
	return &Kustomizer{
		Logger: logger,
		FS:     fs,
		State:  state,
	}
}

func (l *Kustomizer) Execute(ctx context.Context, release *api.Release, step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "execute"))

	current, err := l.State.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	debug.Log("event", "state.loaded")
	kustomizeState := current.CurrentKustomize()

	var shipOverlay state.Overlay
	if kustomizeState == nil {
		debug.Log("event", "state.kustomize.empty")
	} else {
		shipOverlay = kustomizeState.Ship()
	}

	debug.Log("event", "mkdir", "dir", step.Dest)
	err = l.FS.MkdirAll(step.Dest, 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", step.Dest)
		return errors.Wrapf(err, "make dir %s", step.Dest)
	}

	relativePatchPaths, err := l.writePatches(shipOverlay, step.Dest)
	if err != nil {
		return err
	}

	err = l.writeOverlay(step, relativePatchPaths)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

	return nil
}
