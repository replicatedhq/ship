package kustomize

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/pkg/types"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
)

func (l *Kustomizer) PreExecute(ctx context.Context, step api.Step) error {
	// make a folder for this step to render a base into
	tempBase := step.Kustomize.TempRenderPath()
	err := l.FS.MkdirAll(tempBase, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "create dir %q to render kustomize into", tempBase)
	}
	l.renderedUpstream = tempBase

	// Split multi doc yaml first as it will be unmarshalled incorrectly in the following steps
	if err := util.SplitAllKustomize(l.FS, step.Kustomize.Base); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml")
	}

	if err := l.initialKustomizeRun(ctx, *step.Kustomize); err != nil {
		return errors.Wrap(err, "initial kustomize run")
	}

	if err := l.resolveExistingKustomize(ctx, step.Kustomize.Overlay); err != nil {
		return errors.Wrap(err, "resolve existing kustomize")
	}

	return nil
}

func (l *Kustomizer) initialKustomizeRun(ctx context.Context, step api.Kustomize) error {
	if err := l.generateTillerPatches(step); err != nil {
		return errors.Wrap(err, "generate tiller patches")
	}

	built, err := l.kustomizeBuild(constants.DefaultOverlaysPath)
	if err != nil {
		return errors.Wrap(err, "build overlay")
	}

	if err := l.writePostKustomizeFiles(step, built); err != nil {
		return errors.Wrap(err, "write initial kustomized yaml")
	}

	if err := l.writeUpdatedBase(step.Base, built); err != nil {
		return errors.Wrap(err, "replace original yaml")
	}

	return nil
}

// this writes the updated base files into a tempdir that can be used as a preview of the rendered yaml for the UI
// this is done by copying files from the original base to the tempdir and then running replaceOriginal over this
func (l *Kustomizer) writeUpdatedBase(base string, built []util.PostKustomizeFile) error {
	err := util.RecursiveNormalizeCopyKustomize(l.FS, base, l.renderedUpstream)
	if err != nil {
		return errors.Wrapf(err, "copy files from %s to %s", base, l.renderedUpstream)
	}

	err = l.replaceOriginal(l.renderedUpstream, built)
	if err != nil {
		return errors.Wrapf(err, "replace files in %s with kustomized contents", l.renderedUpstream)
	}

	return nil
}

// replace this with code to write the built files into a new dir
// we'll use that dir for everything that expects one resource per file
func (l *Kustomizer) replaceOriginal(base string, built []util.PostKustomizeFile) error {
	builtMap := make(map[util.MinimalK8sYaml]util.PostKustomizeFile)
	for _, builtFile := range built {
		builtMap[builtFile.Minimal] = builtFile
	}

	if err := l.FS.Walk(base, func(targetPath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed to walk base path")
		}

		if !l.shouldAddFileToBase(base, []string{}, targetPath) {
			return nil
		}

		originalFileB, err := l.FS.ReadFile(targetPath)
		if err != nil {
			return errors.Wrap(err, "read original file")
		}

		originalMinimal := util.MinimalK8sYaml{}
		if err := yaml.Unmarshal(originalFileB, &originalMinimal); err != nil {
			return errors.Wrap(err, "unmarshal original")
		}

		if originalMinimal.Kind == "CustomResourceDefinition" {
			// Skip CRDs
			return nil
		}

		initKustomized, exists := builtMap[originalMinimal]
		if !exists {
			// Skip if the file does not have a kustomized equivalent
			return nil
		}

		if err := l.FS.Remove(targetPath); err != nil {
			return errors.Wrap(err, "remove original file")
		}

		initKustomizedB, err := util.MarshalIndent(2, initKustomized.Full)
		if err != nil {
			return errors.Wrap(err, "marshal init kustomized")
		}

		if err := l.FS.WriteFile(targetPath, initKustomizedB, info.Mode()); err != nil {
			return errors.Wrap(err, "write init kustomized file")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "replace original with init kustomized")
	}

	return nil
}

func (l *Kustomizer) resolveExistingKustomize(ctx context.Context, overlayDir string) error {
	exists, err := l.FS.Exists(filepath.Join(overlayDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "check if kustomization exists in %s", overlayDir)
	}

	// no kustomization exists, so no need to update the state
	if !exists {
		return nil
	}

	// read in kustomization yaml
	kustomizationYaml, err := l.FS.ReadFile(filepath.Join(overlayDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "read kustomization exists in %s", overlayDir)
	}

	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(kustomizationYaml, &kustomization)
	if err != nil {
		return errors.Wrapf(err, "unmarshal kustomization yaml from %s", overlayDir)
	}

	currentState, err := l.State.CachedState()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	currentKustomize := currentState.CurrentKustomize()
	if currentKustomize == nil {
		currentKustomize = &state.Kustomize{}
	}
	if currentKustomize.Overlays == nil {
		currentKustomize.Overlays = map[string]state.Overlay{}
	}
	currentOverlay := currentKustomize.Ship()
	fsResources := make(map[string]string)

	for _, kustomizeResource := range kustomization.Resources {
		// read resource referred to by kustomize yaml
		kustomizeResourceContents, err := l.FS.ReadFile(filepath.Join(overlayDir, kustomizeResource))
		if err != nil {
			return errors.Wrapf(err, "read resource %s within %s", kustomizeResource, overlayDir)
		}

		fsResources[string(filepath.Separator)+kustomizeResource] = string(kustomizeResourceContents)
	}
	currentOverlay.Resources = fsResources

	fsPatches := make(map[string]string)

	for _, kustomizePatch := range kustomization.PatchesStrategicMerge {
		// read resource referred to by kustomize yaml
		kustomizePatchContents, err := l.FS.ReadFile(filepath.Join(overlayDir, string(kustomizePatch)))
		if err != nil {
			return errors.Wrapf(err, "read patch %s within %s", kustomizePatch, overlayDir)
		}

		fsPatches[string(filepath.Separator)+string(kustomizePatch)] = string(kustomizePatchContents)
	}
	currentOverlay.Patches = fsPatches

	currentKustomize.Overlays["ship"] = currentOverlay
	err = l.State.SaveKustomize(currentKustomize)
	if err != nil {
		return errors.Wrapf(err, "save updated kustomize")
	}

	return nil
}
