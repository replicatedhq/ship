package daemon

import (
	"fmt"

	"path"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
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

			if step.Render != nil {
				d.hackMaybeRunRenderOnGET(debug, c, step)
			} else if step.Terraform != nil {
				d.hackMaybeRunTerraformOnGET(debug, c, step)
			} else {
				d.hydrateAndSend(daemontypes.NewStep(step), c)
			}

			return
		}
	}

	d.errNotFond(c)
}

func (d *NavcycleRoutes) hackMaybeRunRenderOnGET(debug log.Logger, c *gin.Context, step api.Step) {
	debug.Log("event", "renderStep.get", "msg", "(hack) starting render on GET request")
	// HACK HACK HACK because dex can't redux
	//
	// on get render, automatically treat it like a POST to the render step,
	// that is, start rendering, let the UI poll for status.
	//
	// ideally (maybe?) this can happen on the FE, as soon as render page loads, FE does a POST
	//
	// we check if its in the map, for now only run render if its never been run, or if its already done
	state, err := d.StateManager.TryLoad()
	if err != nil {
		c.AbortWithError(500, errors.Wrap(err, "load state"))
		return
	}
	_, renderAlreadyComplete := state.Versioned().V1.Lifecycle.StepsCompleted[step.Shared().ID]
	progress, ok := d.StepProgress.Load(step.Shared().ID)
	shouldRender := !ok || progress.Detail == `{"status":"success"}` && !renderAlreadyComplete
	if shouldRender {
		d.completeStep(c)
	} else {
		d.hydrateAndSend(daemontypes.NewStep(step), c)
	}
	return
}

func (d *NavcycleRoutes) hackMaybeRunTerraformOnGET(debug log.Logger, c *gin.Context, step api.Step) {
	debug.Log("event", "renderStep.get", "msg", "(hack) starting terraform on GET request")

	state, err := d.StateManager.TryLoad()
	if err != nil {
		c.AbortWithError(500, errors.Wrap(err, "load state"))
		return
	}
	_, renderAlreadyComplete := state.Versioned().V1.Lifecycle.StepsCompleted[step.Shared().ID]
	progress, ok := d.StepProgress.Load(step.Shared().ID)
	shouldRender := !ok || progress.Detail == `{"status":"success"}` && !renderAlreadyComplete
	if shouldRender {
		d.completeStep(c)
	} else {
		d.hydrateAndSend(daemontypes.NewStep(step), c)
	}
	return
}

func (d *NavcycleRoutes) hydrateStep(step daemontypes.Step) (*daemontypes.StepResponse, error) {

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

	if step.HelmValues != nil {
		helmValues := currentState.CurrentHelmValues()
		if helmValues != "" {
			step.HelmValues.Values = helmValues
		} else {
			valuesFileContents, err := d.Fs.ReadFile(path.Join(constants.HelmChartPath, "values.yaml"))
			if err != nil {
				return nil, errors.Wrap(err, "read file values.yaml")
			}
			step.HelmValues.Values = string(valuesFileContents)
		}
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
	progress, hasProgress := d.StepProgress.Load(step.Source.Shared().ID)

	/// JAAAANK
	shouldSkipActions := hasProgress && progress.Detail != `{"status":"success"}`

	if shouldSkipActions {
		return nil
	}

	if step.Message != nil {
		return []daemontypes.Action{
			{ButtonType: "primary", Text: "Confirm", LoadingText: "Confirming", OnClick: daemontypes.ActionRequest{
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
	} else if step.KustomizeIntro != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Next",
				LoadingText: "Next",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/navcycle/step/%s", step.Source.Shared().ID),
					Method: "POST",
					Body:   "",
				},
			},
		}
	} else if step.Kustomize != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Finalize Overlays",
				LoadingText: "Finalizing Overlays",
				OnClick: daemontypes.ActionRequest{
					URI:    fmt.Sprintf("/navcycle/step/%s", step.Source.Shared().ID),
					Method: "POST",
					Body:   "",
				},
			},
		}
	} else if step.Config != nil {
		return []daemontypes.Action{
			{
				ButtonType:  "primary",
				Text:        "Continue to next step",
				LoadingText: "Continue to next step",
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
