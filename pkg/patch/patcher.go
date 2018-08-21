package patch

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kubernetes-sigs/kustomize/pkg/resource"
	k8stypes "github.com/kubernetes-sigs/kustomize/pkg/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

const PATCH_TOKEN = "TO_BE_MODIFIED"

type Patcher interface {
	CreateTwoWayMergePatch(string, string) ([]byte, error)
	MergePatches([]byte, []byte) ([]byte, error)
	ApplyPatch(string, api.Kustomize, string) ([]byte, error)
}

type ShipPatcher struct {
	Logger log.Logger
	FS     afero.Afero
}

func NewShipPatcher(logger log.Logger, fs afero.Afero) Patcher {
	return &ShipPatcher{
		Logger: logger,
		FS:     fs,
	}
}

func (p *ShipPatcher) newKubernetesResource(in []byte) (*resource.Resource, error) {
	var out unstructured.Unstructured

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(in), 1024)
	err := decoder.Decode(&out)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}

	return resource.NewResourceFromUnstruct(out), nil
}

func (p *ShipPatcher) writeHeaderToPatch(originalJSON, patchJSON []byte) ([]byte, error) {
	original := map[string]interface{}{}
	patch := map[string]interface{}{}

	err := json.Unmarshal(originalJSON, &original)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal original json")
	}

	err = json.Unmarshal(patchJSON, &patch)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal patch json")
	}

	originalAPIVersion, ok := original["apiVersion"]
	if !ok {
		return nil, errors.New("no apiVersion key present in original")
	}

	originalKind, ok := original["kind"]
	if !ok {
		return nil, errors.New("no kind key present in original")
	}

	originalMetadata, ok := original["metadata"]
	if !ok {
		return nil, errors.New("no metadata key present in original")
	}

	patch["apiVersion"] = originalAPIVersion
	patch["kind"] = originalKind
	patch["metadata"] = originalMetadata

	modifiedPatch, err := json.Marshal(patch)
	if err != nil {
		return nil, errors.Wrap(err, "marshal modified patch json")
	}

	return modifiedPatch, nil
}

func (p *ShipPatcher) CreateTwoWayMergePatch(original, modified string) ([]byte, error) {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "createTwoWayMergePatch"))

	debug.Log("event", "convert.original")
	originalJSON, err := yaml.YAMLToJSON([]byte(original))
	if err != nil {
		return nil, errors.Wrap(err, "convert original file to json")
	}

	debug.Log("event", "convert.modified")
	modifiedJSON, err := yaml.YAMLToJSON([]byte(modified))
	if err != nil {
		return nil, errors.Wrap(err, "convert modified file to json")
	}

	debug.Log("event", "createKubeResource.original")
	r, err := p.newKubernetesResource(originalJSON)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	versionedObj, err := scheme.Scheme.New(r.Id().Gvk())
	if err != nil {
		return nil, errors.Wrap(err, "read group, version kind from kube resource")
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "create two way merge patch")
	}

	modifiedPatchJSON, err := p.writeHeaderToPatch(originalJSON, patchBytes)
	if err != nil {
		return nil, errors.Wrap(err, "write original header to patch")
	}

	patch, err := yaml.JSONToYAML(modifiedPatchJSON)
	if err != nil {
		return nil, errors.Wrap(err, "convert merge patch json to yaml")
	}

	return patch, nil
}

func (p *ShipPatcher) MergePatches(currentPatch, newPatch []byte) ([]byte, error) {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "mergePatches"))

	debug.Log("event", "createKubeResource.originalFile")
	currentResource, err := p.newKubernetesResource(currentPatch)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	debug.Log("event", "createKubeResource.originalFile")
	newResource, err := p.newKubernetesResource(newPatch)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	debug.Log("event", "createNewScheme.originalFile")
	versionedObj, err := scheme.Scheme.New(currentResource.Id().Gvk())
	if err != nil {
		return nil, errors.Wrap(err, "create new scheme based on kube resource")
	}

	debug.Log("event", "newPatchMeta")
	lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "create new patch meta")
	}

	debug.Log("event", "mergeStrategicMergeMapPatch")
	outJSON, err := strategicpatch.MergeStrategicMergeMapPatchUsingLookupPatchMeta(lookupPatchMeta, currentResource.Object, newResource.Object)
	if err != nil {
		return nil, errors.Wrap(err, "merging patches")
	}

	debug.Log("event", "marshal.mergedPatches")
	out, err := json.Marshal(outJSON)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal merged patch")
	}

	debug.Log("event", "json.to.yaml")
	patch, err := yaml.JSONToYAML(out)
	if err != nil {
		return nil, errors.Wrap(err, "convert json to yaml")
	}

	return patch, nil
}

func (p *ShipPatcher) ApplyPatch(patch string, step api.Kustomize, resource string) ([]byte, error) {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "applyPatch"))
	defer p.applyPatchCleanup()

	debug.Log("event", "writeFile.tempBaseKustomizationYaml")
	if err := p.writeTempKustomization(step, resource); err != nil {
		return nil, errors.Wrap(err, "create temp base kustomization yaml")
	}
	defer p.deleteTempKustomization(step)

	debug.Log("event", "mkdir.tempApplyOverlayPath")
	if err := p.FS.MkdirAll(constants.TempApplyOverlayPath, 0755); err != nil {
		return nil, errors.Wrap(err, "create temp apply overlay path")
	}

	debug.Log("event", "writeFile.tempPatch")
	if err := p.FS.WriteFile(path.Join(constants.TempApplyOverlayPath, "temp.yaml"), []byte(patch), 0755); err != nil {
		return nil, errors.Wrap(err, "write temp patch overlay")
	}

	debug.Log("event", "relPath")
	relativePathToBases, err := filepath.Rel(constants.TempApplyOverlayPath, constants.KustomizeBasePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find relative path")
	}

	kustomizationYaml := k8stypes.Kustomization{
		Bases:   []string{relativePathToBases},
		Patches: []string{"temp.yaml"},
	}

	kustomizationYamlBytes, err := yaml.Marshal(kustomizationYaml)
	if err != nil {
		return nil, errors.Wrap(err, "marshal kustomization yaml")
	}

	debug.Log("event", "writeFile.tempKustomizationYaml")
	if err := p.FS.WriteFile(path.Join(constants.TempApplyOverlayPath, "kustomization.yaml"), kustomizationYamlBytes, 0755); err != nil {
		return nil, errors.Wrap(err, "write temp kustomization yaml")
	}

	debug.Log("event", "run.kustomizeBuild")
	merged, err := p.RunKustomize(constants.TempApplyOverlayPath)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

// TODO(Robert): Mostly a copy of writeBase in kustomize package, but for writing a temporary kustomization yaml
// with a single base resource to which the patch is being applied. Needs refactor and testing around
// matching the targetPath and resource.
func (p *ShipPatcher) writeTempKustomization(step api.Kustomize, resource string) error {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "writeTempKustomization"))

	tempBaseKustomization := k8stypes.Kustomization{}
	if err := p.FS.Walk(
		step.BasePath,
		func(targetPath string, info os.FileInfo, err error) error {
			if err != nil {
				debug.Log("event", "walk.fail", "path", targetPath)
				return errors.Wrap(err, "failed to walk path")
			}

			relativePath, err := filepath.Rel(step.BasePath, targetPath)
			if err != nil {
				debug.Log("event", "relativepath.fail", "base", step.BasePath, "target", targetPath)
				return errors.Wrap(err, "failed to get relative path")
			}

			if targetPath == resource {
				tempBaseKustomization.Resources = append(tempBaseKustomization.Resources, relativePath)
			}
			return nil
		},
	); err != nil {
		return err
	}

	if len(tempBaseKustomization.Resources) == 0 {
		level.Error(p.Logger).Log("event", "unable to find", "resource", resource)
		return errors.New("Temp base directory is empty - base resource not found")
	}

	marshalled, err := yaml.Marshal(tempBaseKustomization)
	if err != nil {
		return errors.Wrap(err, "marshal base kustomization.yaml")
	}

	// write base kustomization
	name := path.Join(step.BasePath, "kustomization.yaml")
	err = p.FS.WriteFile(name, []byte(marshalled), 0666)
	if err != nil {
		return errors.Wrapf(err, "write file %s", name)
	}
	return nil
}

func (p *ShipPatcher) deleteTempKustomization(step api.Kustomize) error {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "deleteTempKustomization"))

	baseKustomizationPath := path.Join(step.BasePath, "kustomization.yaml")

	debug.Log("event", "remove.tempKustomizationYaml")
	err := p.FS.Remove(baseKustomizationPath)
	if err != nil {
		return errors.Wrap(err, "remove temp base kustomization.yaml")
	}

	return nil
}

func (p *ShipPatcher) applyPatchCleanup() {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "patchCleanup"))

	debug.Log("event", "remove temp directory")
	err := p.FS.RemoveAll(constants.TempApplyOverlayPath)
	if err != nil {
		level.Error(log.With(p.Logger, "clean up"))
	}
}

func (p *ShipPatcher) ModifyField(original string, path []string) ([]byte, error) {
	originalMap := map[string]interface{}{}

	originalJSON, err := yaml.YAMLToJSON([]byte(original))
	if err != nil {
		return nil, errors.Wrap(err, "original yaml to json")
	}

	if err := json.Unmarshal([]byte(originalJSON), &originalMap); err != nil {
		return nil, errors.Wrap(err, "unmarshal original yaml")
	}

	modified, err := p.modifyField(originalMap, []string{}, path)
	if err != nil {
		return nil, errors.Wrap(err, "error modifying value")
	}

	modifiedJSON, err := json.Marshal(modified)
	if err != nil {
		return nil, errors.Wrap(err, "marshal modified json")
	}

	modifiedYAML, err := yaml.JSONToYAML(modifiedJSON)
	if err != nil {
		return nil, errors.Wrap(err, "modified json to yaml")
	}

	return modifiedYAML, nil
}

func (p *ShipPatcher) modifyField(original interface{}, current []string, path []string) (interface{}, error) {
	originalType := reflect.TypeOf(original)
	switch originalType.Kind() {
	case reflect.Map:
		typedOriginal, ok := original.(map[string]interface{})
		modifiedMap := make(map[string]interface{})
		if !ok {
			return nil, errors.New("error asserting map")
		}
		for key, value := range typedOriginal {
			modifiedValue, err := p.modifyField(value, append(current, key), path)
			if err != nil {
				return nil, err
			}
			modifiedMap[key] = modifiedValue
		}
		return modifiedMap, nil
	case reflect.Slice:
		typedOriginal, ok := original.([]interface{})
		modifiedSlice := make([]interface{}, len(typedOriginal))
		if !ok {
			return nil, errors.New("error asserting slice")
		}
		for key, value := range typedOriginal {
			modifiedValue, err := p.modifyField(value, append(current, strconv.Itoa(key)), path)
			if err != nil {
				return nil, err
			}
			modifiedSlice[key] = modifiedValue
		}
		return modifiedSlice, nil
	default:
		for idx := range path {
			if current[idx] != path[idx] {
				return original, nil
			}
		}
		return PATCH_TOKEN, nil
	}
}
