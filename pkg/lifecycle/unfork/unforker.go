package unfork

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/replicatedhq/ship/pkg/util"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizepatch "sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/types"
)

func NewDaemonUnforker(logger log.Logger, daemon daemontypes.Daemon, fs afero.Afero, stateManager state.Manager, patcher patch.Patcher) lifecycle.Unforker {
	return &daemonunforker{
		Unforker: Unforker{
			Logger:  logger,
			FS:      fs,
			State:   stateManager,
			Patcher: patcher,
		},
		Daemon: daemon,
	}
}

// unforker will *try* to pull in the Kustomize libs from kubernetes-sigs/kustomize,
// if not we'll have to fork. for now it just explodes
type daemonunforker struct {
	Unforker
	Daemon daemontypes.Daemon
}

func (l *daemonunforker) Execute(ctx context.Context, release *api.Release, step api.Unfork) error {
	daemonExitedChan := l.Daemon.EnsureStarted(ctx, release)
	err := l.awaitUnforkerSaved(ctx, daemonExitedChan)
	if err != nil {
		return errors.Wrap(err, "ensure daemon \"started\"")
	}

	return l.Unforker.Execute(ctx, release, step)
}

// hack -- get the root path off a render step to tell if we should prefix kustomize outputs
func (l *Unforker) getPotentiallyChrootedFs(release *api.Release) (afero.Afero, error) {
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

func (l *daemonunforker) awaitUnforkerSaved(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "method", "unforker.save.await"))
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
		case <-l.Daemon.UnforkSavedChan():
			debug.Log("event", "unfork.finalized")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "unfork.finalized")
		}
	}
}

func (l *Unforker) writeBase(step api.Unfork) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeBase"))

	currentState, err := l.State.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	currentKustomize := currentState.CurrentKustomize()
	if currentKustomize == nil {
		currentKustomize = &state.Kustomize{}
	}
	shipOverlay := currentKustomize.Ship()

	baseKustomization := types.Kustomization{}
	if err := l.FS.Walk(
		step.UpstreamBase,
		func(targetPath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", targetPath)
				return errors.Wrap(err, "failed to walk path")
			}
			relativePath, err := filepath.Rel(step.UpstreamBase, targetPath)
			if err != nil {
				debug.Log("event", "relativepath.fail", "base", step.UpstreamBase, "target", targetPath)
				return errors.Wrap(err, "failed to get relative path")
			}
			if l.shouldAddFileToBase(shipOverlay.ExcludedBases, relativePath) {
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
	name := path.Join(step.UpstreamBase, "kustomization.yaml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}
	return nil
}

func (l *Unforker) shouldAddFileToBase(excludedBases []string, targetPath string) bool {
	if filepath.Ext(targetPath) != ".yaml" && filepath.Ext(targetPath) != ".yml" {
		return false
	}

	for _, base := range excludedBases {
		basePathWOLeading := strings.TrimPrefix(base, "/")
		if basePathWOLeading == targetPath {
			return false
		}
	}

	return !strings.HasSuffix(targetPath, "kustomization.yaml") &&
		!strings.HasSuffix(targetPath, "Chart.yaml") &&
		!strings.HasSuffix(targetPath, "values.yaml")
}

func (l *Unforker) writePatches(fs afero.Afero, shipOverlay state.Overlay, destDir string) (relativePatchPaths []kustomizepatch.PatchStrategicMerge, err error) {
	patches, err := l.writeFileMap(fs, shipOverlay.Patches, destDir)
	if err != nil {
		return nil, errors.Wrapf(err, "write file map to %s", destDir)
	}
	for _, p := range patches {
		relativePatchPaths = append(relativePatchPaths, kustomizepatch.PatchStrategicMerge(p))
	}
	return
}

func (l *Unforker) writeResources(fs afero.Afero, shipOverlay state.Overlay, destDir string) (relativeResourcePaths []string, err error) {
	return l.writeFileMap(fs, shipOverlay.Resources, destDir)
}

func (l *Unforker) writeFileMap(fs afero.Afero, files map[string]string, destDir string) (paths []string, err error) {
	debug := level.Debug(log.With(l.Logger, "method", "writeResources"))

	var keys []string
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, file := range keys {
		contents := files[file]

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

func (l *Unforker) writeFile(fs afero.Afero, name string, contents string) error {
	debug := level.Debug(log.With(l.Logger, "method", "writeFile"))

	destDir := filepath.Dir(name)

	// make the dir
	err := l.FS.MkdirAll(destDir, 0777)
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

func (l *Unforker) writeOverlay(step api.Unfork, relativePatchPaths []kustomizepatch.PatchStrategicMerge, relativeResourcePaths []string) error {
	// just always make a new kustomization.yaml for now
	kustomization := types.Kustomization{
		Bases: []string{
			filepath.Join("../../", step.UpstreamBase),
		},
		PatchesStrategicMerge: relativePatchPaths,
		Resources:             relativeResourcePaths,
	}

	marshalled, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization.yaml")
	}

	name := path.Join(step.OverlayPath(), "kustomization.yaml")
	err = l.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}

	return nil
}

func (l *Unforker) generatePatchesAndExcludeBases(fs afero.Afero, step api.Unfork, upstreamMap map[util.MinimalK8sYaml]string) (*state.Kustomize, error) {
	debug := level.Debug(log.With(l.Logger, "struct", "unforker", "handler", "generatePatchesAndExcludeBases"))

	kustomize := &state.Kustomize{}
	overlay := kustomize.Ship()

	if err := l.FS.Walk(
		step.ForkedBase,
		func(targetPath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", targetPath)
				return errors.Wrap(err, "walk path")
			}

			// ignore non-yaml
			if filepath.Ext(targetPath) != ".yaml" && filepath.Ext(targetPath) != ".yml" {
				return nil
			}

			if info.Mode().IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(step.ForkedBase, targetPath)
			if err != nil {
				debug.Log("event", "relativepath.fail", "base", step.ForkedBase, "target", targetPath)
				return errors.Wrap(err, "get relative path")
			}

			forkedData, err := fs.ReadFile(targetPath)
			if err != nil {
				return errors.Wrap(err, "read forked")
			}

			forkedResource, err := util.NewKubernetesResource(forkedData)
			if err != nil {
				return errors.Wrapf(err, "create new k8s resource %s", targetPath)
			}

			if _, err := scheme.Scheme.New(forkedResource.Id().Gvk()); err != nil {
				// Ignore all non-k8s resources
				return nil
			}

			forkedMinimal := util.MinimalK8sYaml{}
			if err := yaml.Unmarshal(forkedData, &forkedMinimal); err != nil {
				return errors.Wrap(err, "read forked minimal")
			}

			_, fileName := path.Split(relativePath)
			fileName = string(filepath.Separator) + fileName
			upstreamPath, exists := upstreamMap[forkedMinimal]
			if !exists {
				// If no equivalent upstream file exists, it must be a brand new file.
				overlay.Resources[fileName] = string(forkedData)
				debug.Log("event", "resource.saved", "resource", fileName)
				return nil
			}
			delete(upstreamMap, forkedMinimal)

			upstreamData, err := fs.ReadFile(upstreamPath)
			if err != nil {
				return errors.Wrap(err, "read upstream")
			}

			patch, err := l.Patcher.CreateTwoWayMergePatch(upstreamData, forkedData)
			if err != nil {
				return errors.Wrap(err, "create patch")
			}

			includePatch, err := containsNonGVK(patch)
			if err != nil {
				return errors.Wrap(err, "contains non gvk")
			}

			if includePatch {
				overlay.Patches[fileName] = string(patch)
				if err := l.FS.WriteFile(path.Join(step.OverlayPath(), fileName), patch, 0755); err != nil {
					return errors.Wrap(err, "write overlay")
				}
			}

			return nil
		},
	); err != nil {
		return nil, err
	}

	excludedBases := []string{}
	for _, upstream := range upstreamMap {
		relPathToBase, err := filepath.Rel(constants.KustomizeBasePath, upstream)
		if err != nil {
			return nil, errors.Wrapf(err, "relative path to base %s", upstream)
		}
		excludedBases = append(excludedBases, string(filepath.Separator)+relPathToBase)
	}

	overlay.ExcludedBases = excludedBases

	kustomize.Overlays = map[string]state.Overlay{
		"ship": overlay,
	}

	err := l.State.SaveKustomize(kustomize)
	if err != nil {
		return nil, errors.Wrap(err, "save new state")
	}

	return kustomize, nil
}

func (l *Unforker) mapUpstream(upstreamMap map[util.MinimalK8sYaml]string, upstreamPath string) error {
	isDir, err := l.FS.IsDir(upstreamPath)
	if err != nil {
		return errors.Wrapf(err, "is dir %s", upstreamPath)
	}

	if isDir {
		files, err := l.FS.ReadDir(upstreamPath)
		if err != nil {
			return errors.Wrapf(err, "read dir %s", upstreamPath)
		}

		for _, file := range files {
			if err := l.mapUpstream(upstreamMap, filepath.Join(upstreamPath, file.Name())); err != nil {
				return err
			}
		}
	} else {
		upstreamB, err := l.FS.ReadFile(upstreamPath)
		if err != nil {
			return errors.Wrapf(err, "read file %s", upstreamPath)
		}

		upstreamMinimal := util.MinimalK8sYaml{}
		if err := yaml.Unmarshal(upstreamB, &upstreamMinimal); err != nil {
			return errors.Wrapf(err, "unmarshal file %s", upstreamPath)
		}

		upstreamMap[upstreamMinimal] = upstreamPath
	}

	return nil
}

func containsNonGVK(data []byte) (bool, error) {
	gvk := []string{
		"apiVersion",
		"kind",
		"metadata",
	}

	unmarshalled := make(map[string]interface{})
	err := yaml.Unmarshal(data, &unmarshalled)
	if err != nil {
		return false, errors.Wrap(err, "unmarshal patch")
	}

	keys := make([]string, 0, 0)
	for k := range unmarshalled {
		keys = append(keys, k)
	}

	for key := range keys {
		isGvk := false
		for gvkKey := range gvk {
			if key == gvkKey {
				isGvk = true
			}
		}

		if !isGvk {
			return true, nil
		}
	}

	return false, nil
}
