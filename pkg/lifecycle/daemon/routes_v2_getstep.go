package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
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
		tree, err := d.TreeLoader.LoadTree(step.Kustomize.BasePath)
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

	return result, nil
}
