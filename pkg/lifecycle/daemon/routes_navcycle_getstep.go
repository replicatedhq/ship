package daemon

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

func (d *NavcycleRoutes) getStep(c *gin.Context) {
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

func (d *NavcycleRoutes) hydrateStep(step daemontypes.Step, isCurrent bool) (*daemontypes.StepResponse, error) {

	if step.Kustomize != nil {
		tree, err := d.TreeLoader.LoadTree(step.Kustomize.BasePath)
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
	}

	if progress, ok := d.StepProgress.Load(step.Source.Shared().ID); ok {
		result.Progress = &progress
	}

	actions := d.getActions(result.CurrentStep)
	result.Actions = actions

	return result, nil
}

func (d *NavcycleRoutes) getActions(step daemontypes.Step) []daemontypes.Action {
	progress, ok := d.StepProgress.Load(step.Source.Shared().ID)

	shouldAddActions := ok && progress.Detail != "success"

	if shouldAddActions {
		return nil
	}

	if step.Message != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Confirm",
				LoadingText: "Confirming",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/navcycle/step/%s", step.Source.Shared().ID),
					Method: "POST",
					Body:   "",
				},
			},
		}
	} else if step.HelmIntro != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Get started",
				LoadingText: "Confirming",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/navcycle/step/%s", step.Source.Shared().ID),
					Method: "POST",
					Body:   "",
				},
			},
		}
	} else if step.HelmValues != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Saving",
				LoadingText: "Save",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/helm-values"),
					Method: "POST",
					Body:   "",
				},
			},
			{
				ButtonType:  "popover",
				Text:        "Save & Continue",
				LoadingText: "Saving",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/navcycle/step/%s", step.Source.Shared().ID),
					Method: "POST",
					Body:   "",
				},
			},
		}
	}
	return nil
}
