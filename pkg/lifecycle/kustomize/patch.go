package kustomize

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizepatch "sigs.k8s.io/kustomize/pkg/patch"
	k8stypes "sigs.k8s.io/kustomize/pkg/types"
)

type patchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value,omitempty"`
}

var defaultPatches = []patchOperation{
	{
		Op:   "add",
		Path: "/metadata/labels/heritage",
	},
	{
		Op:   "add",
		Path: "/metadata/labels/chart",
	},
	{
		Op:   "remove",
		Path: "/metadata/labels/heritage",
	},
	{
		Op:   "remove",
		Path: "/metadata/labels/chart",
	},
}

const patchPath = "tiller-patch.json"

// generateTillerPatches writes a kustomization.yaml including JSON6902 patches to remove
// the chart and heritage metadata labels.
func (l *Kustomizer) generateTillerPatches(step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "handler", "generateTillerPatches"))

	debug.Log("event", "mkdir.tempApplyOverlayPath")
	if err := l.FS.MkdirAll(constants.TempApplyOverlayPath, 0755); err != nil {
		return errors.Wrap(err, "create temp apply overlay path")
	}

	patchesB, err := json.Marshal(defaultPatches)
	if err != nil {
		return errors.Wrap(err, "marshal heritage patch")
	}

	if err := l.FS.WriteFile(path.Join(constants.TempApplyOverlayPath, patchPath), patchesB, 0755); err != nil {
		return errors.Wrap(err, "write heritage patch")
	}

	relativePathToBases, err := filepath.Rel(constants.TempApplyOverlayPath, step.Base)
	if err != nil {
		return errors.Wrap(err, "relative path to bases")
	}

	json6902Patches := []kustomizepatch.PatchJson6902{}
	if err := l.FS.Walk(
		step.Base,
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

			if info.Name() == "kustomization.yaml" {
				return nil
			}

			fileB, err := l.FS.ReadFile(targetPath)
			if err != nil {
				return errors.Wrapf(err, "read file %s", targetPath)
			}

			resource, err := util.NewKubernetesResource(fileB)
			if err != nil {
				return errors.Wrapf(err, "create new k8s resource %s", targetPath)
			}

			if _, err := scheme.Scheme.New(resource.Id().Gvk()); err != nil {
				// Ignore all non-k8s resources
				return nil
			}

			json6902Patches = append(json6902Patches, kustomizepatch.PatchJson6902{
				Target: &kustomizepatch.Target{
					Group:     resource.GroupVersionKind().Group,
					Version:   resource.GroupVersionKind().Version,
					Kind:      resource.GroupVersionKind().Kind,
					Namespace: resource.GetNamespace(),
					Name:      resource.GetName(),
				},
				Path: patchPath,
			})

			return nil
		},
	); err != nil {
		return err
	}

	kustomizationYaml := k8stypes.Kustomization{
		Bases:           []string{relativePathToBases},
		PatchesJson6902: json6902Patches,
	}

	kustomizationYamlB, err := yaml.Marshal(kustomizationYaml)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization yaml")
	}

	debug.Log("event", "writeFile.kustomization")
	if err := l.FS.WriteFile(path.Join(constants.TempApplyOverlayPath, "kustomization.yaml"), kustomizationYamlB, 0755); err != nil {
		return errors.Wrap(err, "write temp kustomization")
	}

	return nil
}
