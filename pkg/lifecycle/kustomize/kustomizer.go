package kustomize

import (
	"context"

	"time"

	"path"

	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	ktypes "github.com/kubernetes-sigs/kustomize/pkg/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

func NewKustomizer(
	logger log.Logger,
	daemon daemon.Daemon,
	fs afero.Afero,
	stateManager state.Manager,
) lifecycle.Kustomizer {
	return &kustomizer{
		Logger: logger,
		Daemon: daemon,
		FS:     fs,
		State:  stateManager,
	}
}

// kustomizer will *try* to pull in the Kustomizer libs from kubernetes-sigs/kustomize,
// if not we'll have to fork. for now it just explodes
type kustomizer struct {
	Logger log.Logger
	Daemon daemon.Daemon
	FS     afero.Afero
	State  state.Manager
}

func (l *kustomizer) Execute(ctx context.Context, release api.Release, step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "method", "execute"))

	daemonExitedChan := l.Daemon.EnsureStarted(ctx, &release)

	debug.Log("event", "daemon.started")

	l.Daemon.PushKustomizeStep(ctx, daemon.Kustomize{
		BasePath: step.BasePath,
	})
	debug.Log("event", "step.pushed")

	err := l.awaitKustomizeSaved(ctx, daemonExitedChan)
	debug.Log("event", "kustomize.saved", "err", err)
	if err != nil {
		return errors.Wrap(err, "await save kustomize")
	}

	err = l.writeOutOverlays(ctx, step.Dest)
	if err != nil {
		return errors.Wrap(err, "write overlays")
	}

	return nil
}

func (l *kustomizer) writeOutOverlays(
	ctx context.Context,
	destDir string,
) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeOutOverlays"))
	current, err := l.State.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	debug.Log("event", "state.loaded")
	kustomizeState := current.CurrentKustomize()
	if kustomizeState == nil {
		debug.Log("event", "state.kustomize.empty")
		return nil
	}

	// just always make a new kustomization.yaml for now
	kustomization := ktypes.Kustomization{}
	// get the current overlay settings configured in UI
	shipOverlay := kustomizeState.Ship()

	// make the dir
	err = l.FS.MkdirAll(destDir, 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", destDir)
		return errors.Wrapf(err, "make dir %s", destDir)
	}
	debug.Log("event", "mkdir", "dir", destDir)

	// write the overlay patches, updating kustomization.yaml's patch list
	for file, contents := range shipOverlay.Patches {

		name := path.Join(destDir, file)
		err = l.writePatch(name, destDir, contents)
		if err != nil {
			debug.Log("event", "write", "name", name)
			return errors.Wrapf(err, "write %s", name)
		}
		kustomization.Patches = append(kustomization.Patches, file)
	}

	marshalled, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization.yaml")
	}

	name := path.Join(destDir, "kustomization.yml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}

	return nil
}

func (l *kustomizer) writePatch(name string, destDir string, contents string) error {
	debug := level.Debug(log.With(l.Logger, "method", "writePatch"))

	// make the dir
	err := l.FS.MkdirAll(filepath.Dir(name), 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", destDir)
		return errors.Wrapf(err, "make dir %s", destDir)
	}

	// write the file
	err = l.FS.WriteFile(name, []byte(contents), 0666)
	if err != nil {
		return errors.Wrapf(err, "write patch %s", name)
	}
	debug.Log("event", "patch.written", "patch", name)
	return nil
}

func (l *kustomizer) awaitKustomizeSaved(ctx context.Context, daemonExitedChan chan error) error {
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
