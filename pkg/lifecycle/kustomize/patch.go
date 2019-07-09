package kustomize

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizepatch "sigs.k8s.io/kustomize/pkg/patch"
	k8stypes "sigs.k8s.io/kustomize/pkg/types"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
)

type k8sMetadataLabelsOnly struct {
	Metadata struct {
		Labels map[string]interface{} `yaml:"labels"`
	} `yaml:"metadata"`
}

type patchOperation struct {
	Op        string `json:"op"`
	Path      string `json:"path"`
	Value     string `json:"value,omitempty"`
	writePath string
}

var (
	removeHeritagePatch = patchOperation{
		Op:        "remove",
		Path:      "/metadata/labels/heritage",
		writePath: "heritage-patch.json",
	}
	removeChartPatch = patchOperation{
		Op:        "remove",
		Path:      "/metadata/labels/chart",
		writePath: "chart-patch.json",
	}
)

// generateTillerPatches writes a kustomization.yaml including JSON6902 patches to remove
// the chart and heritage metadata labels.
func (l *Kustomizer) generateTillerPatches(step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "kustomizer", "handler", "generateTillerPatches"))

	debug.Log("event", "mkdir.DefaultOverlaysPath")
	if err := l.FS.MkdirAll(constants.DefaultOverlaysPath, 0755); err != nil {
		return errors.Wrapf(err, "create default overlays path at %s", constants.DefaultOverlaysPath)
	}

	defaultPatches := []patchOperation{removeChartPatch, removeHeritagePatch}
	for idx, defaultPatch := range defaultPatches {
		defaultPatchAsSlice := []patchOperation{defaultPatch}

		patchesB, err := json.Marshal(defaultPatchAsSlice)
		if err != nil {
			return errors.Wrapf(err, "marshal default patch idx %d", idx)
		}

		if err := l.FS.WriteFile(path.Join(constants.DefaultOverlaysPath, defaultPatch.writePath), patchesB, 0755); err != nil {
			return errors.Wrapf(err, "write default patch idx %d", idx)
		}
	}

	relativePathToBases, err := filepath.Rel(constants.DefaultOverlaysPath, step.Base)
	if err != nil {
		return errors.Wrap(err, "relative path to bases")
	}

	var excludedBases []string
	state, err := l.State.CachedState()
	if err != nil {
		return errors.Wrap(err, "load state")
	}
	if state.V1 != nil && state.CurrentKustomize() != nil {
		excludedBases = state.CurrentKustomize().Ship().ExcludedBases
	}

	json6902Patches := []kustomizepatch.Json6902{}
	if err := l.FS.Walk(
		step.Base,
		func(targetPath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", targetPath)
				return errors.Wrap(err, "walk path")
			}

			// this ignores non-k8s resources and things included in the list of excluded bases
			if !l.shouldAddFileToBase(step.Base, excludedBases, targetPath) {
				return nil
			}

			fileB, err := l.FS.ReadFile(targetPath)
			if err != nil {
				return errors.Wrapf(err, "read file %s", targetPath)
			}

			resource, err := util.NewKubernetesResource(fileB)
			if err != nil {
				// Ignore all non-k8s resources
				return nil
			}

			if _, err := scheme.Scheme.New(util.ToGroupVersionKind(resource.Id().Gvk())); err != nil {
				// Ignore all non-k8s resources
				return nil
			}

			fileMetadataOnly := k8sMetadataLabelsOnly{}
			if err := yaml.Unmarshal(fileB, &fileMetadataOnly); err != nil {
				return errors.Wrap(err, "unmarshal k8s metadata only")
			}

			for _, excluded := range excludedBases {
				if info.Name() == excluded {
					// don't add this to defaultPatches
					debug.Log("skipping default patches", info.Name())
					return nil
				}
			}

			for _, defaultPatch := range defaultPatches {
				splitDefaultPath := strings.Split(defaultPatch.Path, "/")
				patchLabel := splitDefaultPath[len(splitDefaultPath)-1]

				if l.hasMetadataLabel(patchLabel, fileMetadataOnly) {
					json6902Patches = append(json6902Patches, kustomizepatch.Json6902{
						Target: &kustomizepatch.Target{
							Gvk:       resource.GetGvk(),
							Namespace: resource.Id().Namespace(),
							Name:      resource.GetName(),
						},
						Path: defaultPatch.writePath,
					})
				}
			}

			return nil
		},
	); err != nil {
		return err
	}

	kustomizationYaml := k8stypes.Kustomization{
		Bases:           []string{relativePathToBases},
		PatchesJson6902: json6902Patches,
	}

	kustomizationYamlB, err := util.MarshalIndent(2, kustomizationYaml)
	if err != nil {
		return errors.Wrap(err, "marshal kustomization yaml")
	}

	debug.Log("event", "writeFile.kustomization")
	if err := l.FS.WriteFile(path.Join(constants.DefaultOverlaysPath, "kustomization.yaml"), kustomizationYamlB, 0755); err != nil {
		return errors.Wrap(err, "write temp kustomization")
	}

	return nil
}

func (l *Kustomizer) hasMetadataLabel(label string, k8sFile k8sMetadataLabelsOnly) bool {
	if k8sFile.Metadata.Labels == nil {
		return false
	}

	return k8sFile.Metadata.Labels[label] != nil
}
