package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
)

func (d *V2Routes) getStep(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "handler", "getStep"))
	debug.Log()

	requestedStep := c.Param("step")

	for _, step := range d.Release.Spec.Lifecycle.V1 {
		stepShared := step.Shared()
		if stepShared.ID == requestedStep {

			if ok := d.maybeAbortDueToMissingRequirement(stepShared.Requires, c, requestedStep); !ok {
				return
			}
			d.hydrateAndSend(daemontypes.NewStep(step), c)
			return
		}
	}

	d.errNotFond(c)
}

func (d *V2Routes) hydrateStep(step daemontypes.Step, isCurrent bool) (*daemontypes.StepResponse, error) {
	if step.Kustomize != nil {
		// TODO(Robert): move this into TreeLoader, duplicated in V1 routes
		currentState, err := d.StateManager.TryLoad()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load state")
		}

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

		tree, err := d.TreeLoader.LoadTree(step.Kustomize.BasePath, kustomize)
		if err != nil {
			return nil, errors.Wrap(err, "daemon.loadTree")
		}
		if err != nil {
			level.Error(d.Logger).Log("event", "loadTree.fail", "err", err)
			return nil, errors.Wrap(err, "load kustomize tree")
		}
		step.Kustomize.Tree = *tree
	}

	currentState, err := d.StateManager.TryLoad()
	if err != nil {
		level.Error(d.Logger).Log("event", "tryLoad,fail", "err", err)
		return nil, errors.Wrap(err, "load state")
	}

	helmValues := currentState.CurrentHelmValues()
	if step.HelmValues != nil && helmValues != "" {
		step.HelmValues.Values = helmValues
	}

	result := &daemontypes.StepResponse{
		CurrentStep: step,
		Phase:       step.Source.ShortName(),
		Actions:     []daemontypes.Action{}, //todo actions
	}

	if progress, ok := d.StepProgress[step.Source.Shared().ID]; ok {
		result.Progress = &progress
	}

	return result, nil
}
