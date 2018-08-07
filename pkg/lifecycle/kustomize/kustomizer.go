package kustomize

import (
	"context"
	"os"
	"strings"

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

	shipOverlay := kustomizeState.Ship()

	debug.Log("event", "mkdir", "dir", step.Dest)
	err = l.FS.MkdirAll(step.Dest, 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", step.Dest)
		return errors.Wrapf(err, "make dir %s", step.Dest)
	}

	patches, err := l.writePatches(shipOverlay, step.Dest)
	if err != nil {
		return err
	}

	err = l.writeOverlay(step, patches)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

	err = l.writeBase(step)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

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

func (l *kustomizer) writePatches(shipOverlay state.Overlay, destDir string) ([]string, error) {
	debug := level.Debug(log.With(l.Logger, "method", "writePatches"))

	var patches []string
	for file, contents := range shipOverlay.Patches {
		name := path.Join(destDir, file)
		err := l.writePatch(name, destDir, contents)
		if err != nil {
			debug.Log("event", "write", "name", name)
			return []string{}, errors.Wrapf(err, "write %s", name)
		}
		patches = append(patches, file)
	}
	return patches, nil
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

func (l *kustomizer) writeOverlay(step api.Kustomize, patches []string) error {
	// just always make a new kustomization.yaml for now
	kustomization := ktypes.Kustomization{
		Bases: []string{
			filepath.Join("../../", step.BasePath),
		},
		Patches: patches,
	}

	marshalled, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization.yaml")
	}

	name := path.Join(step.Dest, "kustomization.yaml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}

	return nil
}

func (l *kustomizer) writeBase(step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeBase"))

	baseKustomization := ktypes.Kustomization{}
	if err := l.FS.Walk(
		step.BasePath,
		func(basePath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", basePath)
				return errors.Wrap(err, "failed to walk path")
			}
			if filepath.Ext(basePath) == ".yaml" {
				absolutePathComponents := strings.Split(basePath, "/")
				relativePathComponents := absolutePathComponents[1:]
				relativePath := strings.Join(relativePathComponents, "/")
				baseKustomization.Resources = append(baseKustomization.Resources, relativePath)
			}
			return nil
		},
	); err != nil {
		return err
	}

	if len(baseKustomization.Resources) == 0 {
		return errors.New("Base directory is empty")
	}

	marshalled, err := yaml.Marshal(baseKustomization)
	if err != nil {
		return errors.Wrap(err, "marshal base kustomization.yaml")
	}

	// write base kustomization
	name := path.Join(step.BasePath, "kustomization.yaml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}
	return nil
}
