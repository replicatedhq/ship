package kustomize

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type Kustomizer struct {
	Logger  log.Logger
	FS      afero.Afero
	State   state.Manager
	Patcher patch.ShipPatcher
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

	debug.Log("event", "write.base.kustomization.yaml")
	err := l.writeBase(step)
	if err != nil {
		return errors.Wrap(err, "write base kustomization")
	}

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

	fs, err := l.getPotentiallyChrootedFs(release)
	if err != nil {
		debug.Log("event", "getFs.fail")
		return errors.Wrapf(err, "get base fs")
	}

	debug.Log("event", "mkdir", "dir", step.OverlayPath())
	err = fs.MkdirAll(step.OverlayPath(), 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", step.OverlayPath())
		return errors.Wrapf(err, "make dir %s", step.OverlayPath())
	}

	relativePatchPaths, err := l.writePatches(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	relativeResourcePaths, err := l.writeResources(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	err = l.writeOverlay(fs, step, relativePatchPaths, relativeResourcePaths)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

	if step.Dest != "" {
		debug.Log("event", "kustomize.build", "dest", step.Dest)
		err = l.kustomizeBuild(fs, step)
		if err != nil {
			return errors.Wrap(err, "build overlay")
		}
	}

	return nil
}
func (l *Kustomizer) kustomizeBuild(fs afero.Afero, kustomize api.Kustomize) error {
	builtYAML, err := l.Patcher.RunKustomize(kustomize.OverlayPath())
	if err != nil {
		return errors.Wrap(err, "run kustomize")
	}

	fs.WriteFile(kustomize.Dest, builtYAML, 0644)
	return nil
}
