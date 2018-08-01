package daemon

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/state"
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
