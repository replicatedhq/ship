package daemon

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/pkg/resource"
)

type SaveOverlayRequest struct {
	Path       string `json:"path"`
	Contents   string `json:"contents"`
	IsResource bool   `json:"isResource"`
}

func (d *NavcycleRoutes) kustomizeSaveOverlay(c *gin.Context) {

	var request SaveOverlayRequest
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		return
	}

	if request.Path == "" || request.Contents == "" {
		c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error":  "bad_request",
				"detail": "Patch and resource contents cannot be empty",
			},
		)
		return
	}

	step, ok := d.getKustomizeStepOrAbort(c)
	if !ok {
		return
	}

	if err := d.kustomizeDoSaveOverlay(request); err != nil {
		level.Error(d.Logger).Log("event", "saveOverlay.fail", "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	level.Debug(d.Logger).Log("event", "stepProgress.storeStatus")
	d.StepProgress.Delete(step.Shared().ID)

	c.JSON(200, map[string]string{"status": "success"})
}

func (d *NavcycleRoutes) kustomizeDoSaveOverlay(request SaveOverlayRequest) error {
	debug := level.Debug(log.With(d.Logger, "handler", "kustomizeSaveOverlay"))

	debug.Log("event", "state.load")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	debug.Log("event", "current.load")
	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	if kustomize.Overlays == nil {
		kustomize.Overlays = map[string]state.Overlay{}
	}

	overlay := kustomize.Ship()

	if request.IsResource {
		if overlay.Resources == nil {
			overlay.Resources = map[string]string{}
		}
		overlay.Resources[request.Path] = request.Contents
	} else {
		if overlay.Patches == nil {
			overlay.Patches = map[string]string{}
		}
		overlay.Patches[request.Path] = request.Contents
	}

	kustomize.Overlays["ship"] = overlay

	debug.Log("event", "newstate.save")
	err = d.StateManager.SaveKustomize(kustomize)
	if err != nil {
		return errors.Wrap(err, "save new state")
	}

	return nil
}

// TODO(Robert): duped logic in filetree
func isSupported(file []byte) bool {
	var out unstructured.Unstructured

	fileJSON, err := yaml.YAMLToJSON(file)
	if err != nil {
		return false
	}

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(fileJSON), 1024)
	if err := decoder.Decode(&out); err != nil {
		return false
	}

	r := resource.NewResourceFromUnstruct(out)
	if r.GetKind() == "CustomResourceDefinition" {
		return false
	}

	return true
}

func (d *NavcycleRoutes) kustomizeGetFile(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "kustomizeGetFile"))
	debug.Log()

	type Request struct {
		Path string `json:"path"`
	}

	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		return
	}

	type Response struct {
		Base        string `json:"base"`
		IsSupported bool   `json:"isSupported"`
		IsResource  bool   `json:"isResource"`
		Overlay     string `json:"overlay"`
	}

	step, ok := d.getKustomizeStepOrAbort(c) // todo this should fetch by step ID
	if !ok {
		return
	}

	savedState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "load state failed", "err", err)
		c.AbortWithError(500, err)
		return
	}

	overlay, isResource := savedState.CurrentKustomizeOverlay(request.Path)

	var base []byte
	if !isResource {
		base, err = d.TreeLoader.LoadFile(step.Kustomize.Base, request.Path)
		if err != nil {
			level.Warn(d.Logger).Log("event", "load file failed", "err", err)
			c.AbortWithError(500, err)
			return
		}
	}

	c.JSON(200, Response{
		Base:        string(base),
		Overlay:     overlay,
		IsResource:  isResource,
		IsSupported: isSupported(base),
	})
}

func (d *NavcycleRoutes) getKustomizeStepOrAbort(c *gin.Context) (api.Step, bool) {
	var step api.Step
	for _, step = range d.Release.Spec.Lifecycle.V1 {
		if step.Kustomize != nil {
			ok := d.maybeAbortDueToMissingRequirement(step.Shared().Requires, c, step.Shared().ID)
			return step, ok
		}
	}
	return step, false
}

func (d *NavcycleRoutes) kustomizeFinalize(c *gin.Context) {
	level.Debug(d.Logger).Log("event", "kustomize.finalize", "detail", "not implemented")
	c.JSON(200, map[string]interface{}{"status": "success"})
}

func (d *NavcycleRoutes) applyPatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "applyPatch"))
	type Request struct {
		Resource string `json:"resource"`
		Patch    string `json:"patch"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		return
	}

	debug.Log("event", "getKustomizationStep")
	step, ok := d.getKustomizeStepOrAbort(c)
	if !ok {
		level.Error(d.Logger).Log("event", "get kustomize step")
		return
	}

	modified, err := d.Patcher.ApplyPatch([]byte(request.Patch), *step.Kustomize, request.Resource)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to merge patch with base", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}

	c.JSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}

func (d *NavcycleRoutes) createOrMergePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "createOrMergePatch"))
	type Request struct {
		Original string        `json:"original"`
		Current  string        `json:"current"`
		Path     []interface{} `json:"path"`
		Resource string        `json:"resource"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		return
	}

	var stringPath []string
	for _, value := range request.Path {
		switch value.(type) {
		case float64:
			stringPath = append(stringPath, strconv.FormatFloat(value.(float64), 'f', 0, 64))
		case string:
			stringPath = append(stringPath, value.(string))
		default:
			level.Error(d.Logger).Log("event", "invalid path provided")
			c.AbortWithError(500, errors.New("internal_server_error"))
			return
		}
	}

	step, ok := d.getKustomizeStepOrAbort(c)
	if !ok {
		return
	}

	debug.Log("event", "load.originalFile")
	original, err := d.TreeLoader.LoadFile(step.Kustomize.Base, request.Original)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to read original file", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}

	debug.Log("event", "patcher.modifyField")
	modified, err := d.Patcher.ModifyField(original, stringPath)
	if err != nil {
		level.Error(d.Logger).Log("event", "modify field", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}

	debug.Log("event", "patcher.CreatePatch")
	patch, err := d.Patcher.CreateTwoWayMergePatch(original, modified)
	if err != nil {
		level.Error(d.Logger).Log("event", "create two way merge patch", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}

	if len(request.Current) > 0 {
		out, err := d.Patcher.MergePatches([]byte(request.Current), stringPath, *step.Kustomize, request.Resource)
		if err != nil {
			level.Error(d.Logger).Log("event", "merge current and new patch", "err", err)
			c.AbortWithError(500, errors.New("internal_server_error"))
			return
		}
		c.JSON(200, map[string]interface{}{
			"patch": string(out),
		})
	} else {
		c.JSON(200, map[string]interface{}{
			"patch": string(patch),
		})
	}
}

func (d *NavcycleRoutes) deleteBase(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deleteBase"))
	pathQueryParam := c.Query("path")
	if pathQueryParam == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "delete base"))
		return
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	shipOverlay := kustomize.Ship()
	for _, base := range shipOverlay.ExcludedBases {
		if base == pathQueryParam {
			debug.Log("event", "base", pathQueryParam, "exists in excluded")
			c.AbortWithError(http.StatusInternalServerError, errors.New("internal_server_error"))
			return
		}
	}
	shipOverlay.ExcludedBases = append(shipOverlay.ExcludedBases, pathQueryParam)

	if _, exists := shipOverlay.Patches[pathQueryParam]; exists {
		delete(shipOverlay.Patches, pathQueryParam)
	}

	if kustomize.Overlays == nil {
		kustomize.Overlays = map[string]state.Overlay{}
	}
	kustomize.Overlays["ship"] = shipOverlay

	if err := d.StateManager.SaveKustomize(kustomize); err != nil {
		c.AbortWithError(500, errors.Wrap(err, "delete base"))
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}

func (d *NavcycleRoutes) includeBase(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "includeBase"))
	type includeRequest struct {
		Path string `json:"path"`
	}
	var request includeRequest
	debug.Log("event", "unmarshal request")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		return
	}

	debug.Log("event", "load state")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "delete base"))
		return
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	shipOverlay := kustomize.Ship()
	newExcludedBases := make([]string, 0)
	for _, base := range shipOverlay.ExcludedBases {
		if base != request.Path {
			newExcludedBases = append(newExcludedBases, base)
		}
	}
	shipOverlay.ExcludedBases = newExcludedBases

	if kustomize.Overlays == nil {
		kustomize.Overlays = map[string]state.Overlay{}
	}
	kustomize.Overlays["ship"] = shipOverlay

	if err := d.StateManager.SaveKustomize(kustomize); err != nil {
		c.AbortWithError(500, errors.Wrap(err, "delete base"))
		return
	}
	c.JSON(200, map[string]string{"status": "success"})
}

func (d *NavcycleRoutes) deleteResource(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deleteResource"))
	pathQueryParam := c.Query("path")
	if pathQueryParam == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	debug.Log("event", "resource.delete", "path", pathQueryParam)
	err := d.deleteFile(pathQueryParam, func(overlay state.Overlay) map[string]string {
		return overlay.Resources
	})

	if err != nil {
		level.Error(d.Logger).Log("event", "resource.delete.fail", "path", pathQueryParam, "err", err)
		c.AbortWithError(500, errors.Wrap(err, "delete resource"))
		return
	}
	c.JSON(200, map[string]string{"status": "success"})
}

func (d *NavcycleRoutes) deletePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deleteResource"))
	pathQueryParam := c.Query("path")
	if pathQueryParam == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	debug.Log("event", "resource.delete", "path", pathQueryParam)
	err := d.deleteFile(pathQueryParam, func(overlay state.Overlay) map[string]string {
		return overlay.Patches
	})

	if err != nil {
		level.Error(d.Logger).Log("event", "resource.delete.fail", "path", pathQueryParam, "err", err)
		c.AbortWithError(500, errors.Wrap(err, "delete resource"))
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}

func (d *NavcycleRoutes) deleteFile(pathQueryParam string, getFiles func(overlay state.Overlay) map[string]string) error {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deleteFile"))
	debug.Log("event", "state.load")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		return errors.New("current kustomize empty")
	}

	shipOverlay := kustomize.Ship()
	files := getFiles(shipOverlay)

	if len(files) == 0 {
		return errors.New("no files to delete")
	}

	_, ok := files[pathQueryParam]
	if !ok {
		return errors.New("not found: file not in map")
	}

	debug.Log("event", "deletePatch", "path", pathQueryParam)
	delete(files, pathQueryParam)

	if shipOverlay.Patches == nil && shipOverlay.Resources == nil {
		kustomize.Overlays["ship"] = state.NewOverlay()
	}

	if err := d.StateManager.SaveKustomize(kustomize); err != nil {
		return errors.Wrap(err, "save updated kustomize")
	}
	return nil
}
