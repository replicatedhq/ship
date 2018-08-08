package patch

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kubernetes-sigs/kustomize/pkg/resource"
	k8stypes "github.com/kubernetes-sigs/kustomize/pkg/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/process"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

type Patcher interface {
	CreateTwoWayMergePatch(string, string) ([]byte, error)
	MergePatches([]byte, []byte) ([]byte, error)
	ApplyPatch(string) ([]byte, error)
}

type ShipPatcher struct {
	Logger  log.Logger
	FS      afero.Afero
	process process.Process
}

func NewShipPatcher(logger log.Logger, fs afero.Afero) Patcher {
	process := process.Process{Logger: logger}

	return &ShipPatcher{
		Logger:  logger,
		FS:      fs,
		process: process,
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

func (p *ShipPatcher) ApplyPatch(patch string) ([]byte, error) {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "applyPatch"))
	defer p.applyPatchCleanup()

	kustomizeCmd := exec.Command("/usr/local/bin/kustomize")

	debug.Log("event", "mkdir.tempApplyOverlayPath")
	if err := p.FS.MkdirAll(constants.TempApplyOverlayPath, 0755); err != nil {
		return nil, errors.Wrap(err, "create temp apply overlay path")
	}

	debug.Log("event", "writeFile.tempPatch")
	if err := p.FS.WriteFile(path.Join(constants.TempApplyOverlayPath, "temp.yaml"), []byte(patch), 0755); err != nil {
		return nil, errors.Wrap(err, "write temp patch overlay")
	}

	debug.Log("event", "relPath")
	relativePathToBases, err := filepath.Rel(constants.TempApplyOverlayPath, constants.RenderedHelmPath)
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

	kustomizeCmd.Args = append(
		kustomizeCmd.Args,
		"build", constants.TempApplyOverlayPath,
	)

	debug.Log("event", "fork")
	stdout, stderr, err := p.process.Fork(kustomizeCmd)
	if err != nil {
		debug.Log("event", "cmd.err")
		if exitError, ok := err.(*exec.ExitError); ok && !exitError.Success() {
			return nil, errors.Errorf(`execute kustomize: %s: stdout: "%s"; stderr: "%s";`, exitError.Error(), stdout, stderr)
		}
		return nil, errors.Wrap(err, "execute kustomize")
	}
	return stdout, nil
}

func (p *ShipPatcher) applyPatchCleanup() {
	debug := level.Debug(log.With(p.Logger, "struct", "patcher", "handler", "patchCleanup"))

	debug.Log("event", "remove temp directory")
	err := p.FS.RemoveAll(constants.TempApplyOverlayPath)
	if err != nil {
		level.Error(log.With(p.Logger, "clean up"))
	}
}
