package daemon

import (
	"bytes"
	"context"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kubernetes-sigs/kustomize/pkg/resource"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/state"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/apimachinery/pkg/util/json"
)

func (d *ShipDaemon) requireKustomize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.currentStep == nil || d.currentStep.Kustomize == nil {
			c.AbortWithError(
				400,
				errors.Errorf("bad request: expected phase kustomize, was %q", d.currentStepName),
			)
		}
		c.Next()

	}
}

func (d *ShipDaemon) KustomizeSavedChan() chan interface{} {
	return d.KustomizeSaved
}

func (d *ShipDaemon) PushKustomizeStep(ctx context.Context, kustomize Kustomize) {
	debug := level.Debug(log.With(d.Logger, "method", "PushKustomizeStep"))
	defer d.locker(debug)()
	d.cleanPreviousStep()

	d.currentStepName = StepNameKustomize
	d.currentStep = &Step{Kustomize: &kustomize}
	d.KustomizeSaved = make(chan interface{}, 1)
	d.NotifyStepChanged(StepNameKustomize)
}

func (d *ShipDaemon) kustomizeSaveOverlay(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "kustomizeSaveOverlay"))
	defer d.locker(debug)()
	type Request struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
	}

	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	debug.Log("event", "request.bind")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	debug.Log("event", "current.load")
	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	if kustomize.Overlays == nil {
		kustomize.Overlays = make(map[string]state.Overlay)
	}

	if _, ok := kustomize.Overlays["ship"]; !ok {
		kustomize.Overlays["ship"] = state.Overlay{
			Patches: make(map[string]string),
		}
	}

	kustomize.Overlays["ship"].Patches[request.Path] = request.Contents

	debug.Log("event", "newstate.save")
	err = d.StateManager.SaveKustomize(kustomize)
	if err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}

func (d *ShipDaemon) kustomizeGetFile(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "kustomizeGetFile"))
	defer d.locker(debug)()

	type Request struct {
		Path string `json:"path"`
	}

	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	type Response struct {
		Base    string `json:"base"`
		Overlay string `json:"overlay"`
	}
	base, err := d.TreeLoader.LoadFile(d.currentStep.Kustomize.BasePath, request.Path)
	if err != nil {
		level.Warn(d.Logger).Log("event", "load file failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	savedState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "load state failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, Response{
		Base:    base,
		Overlay: savedState.CurrentKustomizeOverlay(request.Path),
	})
}

func (d *ShipDaemon) kustomizeFinalize(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "kustomizeFinalize"))
	defer d.locker(debug)()

	level.Debug(d.Logger).Log("event", "kustomize.finalize", "detail", "not implemented")
	d.KustomizeSaved <- nil
	c.JSON(200, map[string]interface{}{"status": "success"})
}
func (d *ShipDaemon) loadKustomizeTree() (*filetree.Node, error) {
	level.Debug(d.Logger).Log("event", "kustomize.loadTree")
	tree, err := d.TreeLoader.LoadTree(d.currentStep.Kustomize.BasePath)
	if err != nil {
		return nil, errors.Wrap(err, "daemon.loadTree")
	}
	return tree, nil
}

func (d *ShipDaemon) newKubernetesResource(in []byte) (*resource.Resource, error) {
	var out unstructured.Unstructured

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(in), 1024)
	err := decoder.Decode(&out)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}

	return resource.NewResourceFromUnstruct(out), nil
}

func (d *ShipDaemon) writeHeaderToPatch(originalJSON, patchJSON []byte) ([]byte, error) {
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

func (d *ShipDaemon) createTwoWayMergePatch(originalFilePath, modified string) ([]byte, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "createTwoWayMergePatch"))

	debug.Log("event", "load.originalFile")
	originalString, err := d.TreeLoader.LoadFile(d.currentStep.Kustomize.BasePath, originalFilePath)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to read original file", "err", err)
	}

	debug.Log("event", "convert.originalFile")
	originalJSON, err := yaml.YAMLToJSON([]byte(originalString))
	if err != nil {
		return nil, errors.Wrap(err, "convert original file to json")
	}

	debug.Log("event", "convert.modifiedFile")
	modifiedJSON, err := yaml.YAMLToJSON([]byte(modified))
	if err != nil {
		return nil, errors.Wrap(err, "convert modified file to json")
	}

	debug.Log("event", "createKubeResource.originalFile")
	r, err := d.newKubernetesResource(originalJSON)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	versionedObj, _ := scheme.Scheme.New(r.Id().Gvk())

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "create two way merge patch")
	}

	modifiedPatchJSON, err := d.writeHeaderToPatch(originalJSON, patchBytes)
	if err != nil {
		return nil, errors.Wrap(err, "write original header to patch")
	}

	patch, err := yaml.JSONToYAML(modifiedPatchJSON)
	if err != nil {
		return nil, errors.Wrap(err, "convert merge patch json to yaml")
	}

	return patch, nil
}

func (d* ShipDaemon) mergePatches(originalFilePath string, currentPatch, newPatch []byte) ([]byte, error) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "mergePatches"))

	debug.Log("event", "createKubeResource.originalFile")
	currentResource, err := d.newKubernetesResource(currentPatch)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	debug.Log("event", "createKubeResource.originalFile")
	newResource, err := d.newKubernetesResource(newPatch)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	debug.Log("event", "createNewScheme.originalFile")
	versionedObj, err := scheme.Scheme.New(currentResource.Id().Gvk())
	if err != nil {
		return nil, errors.Wrap(err, "create new scheme based on kube resource")
	}

	lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "create new patch meta")
	}

	debug.Log("event", "mergeStrategicMergeMapPatch")
	outJSON, err := strategicpatch.MergeStrategicMergeMapPatchUsingLookupPatchMeta(lookupPatchMeta, currentResource.Object, newResource.Object)
	if err != nil {
		return nil, errors.Wrap(err, "merging patches")
	}

	out, err := json.Marshal(outJSON)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal merged patch")
	}

	patch, err := yaml.JSONToYAML(out)
	if err != nil {
		return nil, errors.Wrap(err, "convert json to yaml")
	}

	return patch, nil
}
