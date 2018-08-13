package daemon

import (
	"fmt"

	"path"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
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

			if step.Render == nil {
				d.hydrateAndSend(daemontypes.NewStep(step), c)
				return
			} else {
				debug.Log("event", "renderStep.get", "msg", "(hack) starting render on GET request")
				// HACK HACK HACK because dex can't redux
				//
				// on get render, automatically treat it like a POST to the render step,
				// that is, start rendering, let the UI poll for status.
				//
				// ideally (maybe?) this can happen on the FE, as soon as render page loads, FE does a POST
				//
				// we check if its in the map, for now only run render if its never been run, or if its already done
				progress, ok := d.StepProgress.Load(step.Shared().ID)
				if !ok || progress.Detail == "success" {
					d.completeStep(c)
				} else {
					d.hydrateAndSend(daemontypes.NewStep(step), c)
				}
				return
			}
		}
	}

	d.errNotFond(c)
}

func (d *NavcycleRoutes) hydrateStep(step daemontypes.Step) (*daemontypes.StepResponse, error) {

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

	if step.HelmValues != nil {
		helmValues := currentState.CurrentHelmValues()
		if helmValues != "" {
			step.HelmValues.Values = helmValues
		} else {
			valuesFileContents, err := d.Fs.ReadFile(path.Join(constants.KustomizeHelmPath, "values.yaml"))
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
	progress, ok := d.StepProgress.Load(step.Source.Shared().ID)

	shouldAddActions := ok && progress.Detail != "success"

	if shouldAddActions {
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
	}
	return nil
}
