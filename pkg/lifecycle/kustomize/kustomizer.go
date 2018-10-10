package kustomize

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/pkg/patch"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
)

func NewDaemonKustomizer(
	logger log.Logger,
	daemon daemontypes.Daemon,
	fs afero.Afero,
	stateManager state.Manager,
) lifecycle.Kustomizer {
	return &daemonkustomizer{
		Kustomizer: Kustomizer{
			Logger: logger,
			FS:     fs,
			State:  stateManager,
		},
		Daemon: daemon,
	}
}

// kustomizer will *try* to pull in the Kustomizer libs from kubernetes-sigs/kustomize,
// if not we'll have to fork. for now it just explodes
type daemonkustomizer struct {
	Kustomizer
	Daemon daemontypes.Daemon
}

func (l *daemonkustomizer) Execute(ctx context.Context, release *api.Release, step api.Kustomize) error {
	daemonExitedChan := l.Daemon.EnsureStarted(ctx, release)
	err := l.awaitKustomizeSaved(ctx, daemonExitedChan)
	if err != nil {
		return errors.Wrap(err, "ensure daemon \"started\"")
	}

	return l.Kustomizer.Execute(ctx, release, step)
}

// hack -- get the root path off a render step to tell if we should prefix kustomize outputs
func (l *Kustomizer) getPotentiallyChrootedFs(release *api.Release) (afero.Afero, error) {
	renderRoot := constants.InstallerPrefixPath
	renderStep := release.FindRenderStep()
	if renderStep == nil || renderStep.Root == "./" || renderStep.Root == "." {
		return l.FS, nil
	}
	if renderStep.Root != "" {
		renderRoot = renderStep.Root
	}

	fs := afero.Afero{Fs: afero.NewBasePathFs(l.FS, renderRoot)}
	err := fs.MkdirAll("/", 0755)
	if err != nil {
		return afero.Afero{}, errors.Wrap(err, "mkdir fs root")
	}
	return fs, nil
}

func (l *daemonkustomizer) awaitKustomizeSaved(ctx context.Context, daemonExitedChan chan error) error {
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

func (l *Kustomizer) writePatches(
	fs afero.Afero,
	shipOverlay state.Overlay,
	destDir string,
) (relativePatchPaths []patch.PatchStrategicMerge, err error) {
	patches, err := l.writeFileMap(fs, shipOverlay.Patches, destDir)
	if err != nil {
		return nil, errors.Wrapf(err, "write file map to %s", destDir)
	}
	for _, p := range patches {
		relativePatchPaths = append(relativePatchPaths, patch.PatchStrategicMerge(p))
	}
	return
}

func (l *Kustomizer) writeResources(fs afero.Afero, shipOverlay state.Overlay, destDir string) (relativeResourcePaths []string, err error) {
	return l.writeFileMap(fs, shipOverlay.Resources, destDir)
}

func (l *Kustomizer) writeFileMap(fs afero.Afero, files map[string]string, destDir string) (paths []string, err error) {
	debug := level.Debug(log.With(l.Logger, "method", "writeResources"))

	for file, contents := range files {
		name := path.Join(destDir, file)
		err := l.writeFile(fs, name, contents)
		if err != nil {
			debug.Log("event", "write", "name", name)
			return []string{}, errors.Wrapf(err, "write resource %s", name)
		}

		relativePatchPath, err := filepath.Rel(destDir, name)
		if err != nil {
			return []string{}, errors.Wrap(err, "unable to determine relative path")
		}
		paths = append(paths, relativePatchPath)
	}
	return paths, nil

}

func (l *Kustomizer) writeFile(fs afero.Afero, name string, contents string) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeFile"))

	destDir := filepath.Dir(name)

	// make the dir
	err := fs.MkdirAll(destDir, 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", destDir)
		return errors.Wrapf(err, "make dir %s", destDir)
	}

	// write the file
	err = fs.WriteFile(name, []byte(contents), 0666)
	if err != nil {
		return errors.Wrapf(err, "write patch %s", name)
	}
	debug.Log("event", "patch.written", "patch", name)
	return nil
}

func (l *Kustomizer) writeOverlay(
	fs afero.Afero,
	step api.Kustomize,
	relativePatchPaths []patch.PatchStrategicMerge,
	relativeResourcePaths []string,
) error {
	// just always make a new kustomization.yaml for now
	kustomization := ktypes.Kustomization{
		Bases: []string{
			filepath.Join("../../", step.Base),
		},
		PatchesStrategicMerge: relativePatchPaths,
		Resources:             relativeResourcePaths,
	}

	marshalled, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization.yaml")
	}

	name := path.Join(step.OverlayPath(), "kustomization.yaml")
	err = fs.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}

	return nil
}

func (l *Kustomizer) writeBase(step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeBase"))

	baseKustomization := ktypes.Kustomization{}
	if err := l.FS.Walk(
		step.Base,
		func(targetPath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", targetPath)
				return errors.Wrap(err, "failed to walk path")
			}
			if l.shouldAddFile(targetPath) {
				relativePath, err := filepath.Rel(step.Base, targetPath)
				if err != nil {
					debug.Log("event", "relativepath.fail", "base", step.Base, "target", targetPath)
					return errors.Wrap(err, "failed to get relative path")
				}
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
	name := path.Join(step.Base, "kustomization.yaml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}
	return nil
}

func (l *Kustomizer) shouldAddFile(targetPath string) bool {
	return filepath.Ext(targetPath) == ".yaml" &&
		!strings.HasSuffix(targetPath, "kustomization.yaml") &&
		!strings.HasSuffix(targetPath, "Chart.yaml") &&
		!strings.HasSuffix(targetPath, "values.yaml")
}
