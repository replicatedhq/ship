package daemon

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
)

func (d *NavcycleRoutes) kustomizeSaveOverlay(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "kustomizeSaveOverlay"))
	type Request struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
	}

	var request Request
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if request.Path == "" || request.Contents == "" {
		c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error":  "bad_request",
				"detail": "path and contents cannot be empty",
			},
		)
		return
	}

	step, ok := d.getKustomizeStepOrAbort(c)
	if !ok {
		return
	}

	debug.Log("event", "request.bind")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
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

	debug.Log("event", "stepProgress.storeStatus")
	d.StepProgress.Delete(step.Shared().ID)

	c.JSON(200, map[string]string{"status": "success"})
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
		c.AbortWithError(500, err)
		return
	}

	type Response struct {
		Base    string `json:"base"`
		Overlay string `json:"overlay"`
	}

	step, ok := d.getKustomizeStepOrAbort(c) // todo this should fetch by step ID
	if !ok {
		return
	}

	base, err := d.TreeLoader.LoadFile(step.Kustomize.BasePath, request.Path)
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
		Patch string `json:"patch"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	modified, err := d.Patcher.ApplyPatch(request.Patch)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to merge patch with base", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	c.JSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}

func (d *NavcycleRoutes) createOrMergePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "createOrMergePatch"))
	type Request struct {
		Original string `json:"original"`
		Modified string `json:"modified"`
		Current  string `json:"current"`
	}
	var request Request

	debug.Log("event", "request.bind")
	if err := c.BindJSON(&request); err != nil {
		level.Error(d.Logger).Log("event", "unmarshal request body failed", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	step, ok := d.getKustomizeStepOrAbort(c)
	if !ok {
		return
	}

	debug.Log("event", "load.originalFile")
	original, err := d.TreeLoader.LoadFile(step.Kustomize.BasePath, request.Original)
	if err != nil {
		level.Error(d.Logger).Log("event", "failed to read original file", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	debug.Log("event", "patcher.CreatePatch")
	patch, err := d.Patcher.CreateTwoWayMergePatch(original, request.Modified)
	if err != nil {
		level.Error(d.Logger).Log("event", "create two way merge patch", "err", err)
		c.AbortWithError(500, errors.New("internal_server_error"))
	}

	if request.Current != "" {
		out, err := d.Patcher.MergePatches([]byte(request.Current), patch)
		if err != nil {
			level.Error(d.Logger).Log("event", "merge current and new patch", "err", err)
			c.AbortWithError(500, errors.New("internal_server_error"))
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

func (d *NavcycleRoutes) deletePatch(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "struct", "daemon", "handler", "deletePatch"))
	pathQueryParam := c.Query("path")
	if pathQueryParam == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
	}

	debug.Log("event")
	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "try load state failed", "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		level.Error(d.Logger).Log("event", "empty kustomize")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	shipOverlay := kustomize.Ship()
	if len(shipOverlay.Patches) == 0 {
		level.Error(d.Logger).Log("event", "empty ship overlay")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	_, ok := shipOverlay.Patches[pathQueryParam]
	if !ok {
		level.Error(d.Logger).Log("event", "patch does not exist")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	debug.Log("event", "deletePatch", "path", pathQueryParam)
	delete(shipOverlay.Patches, pathQueryParam)

	if err := d.StateManager.SaveKustomize(kustomize); err != nil {
		level.Error(d.Logger).Log("event", "patch does not exist")
		c.AbortWithError(http.StatusBadRequest, errors.New("bad delete request"))
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}
