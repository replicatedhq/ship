package daemon

import (
	"context"
	"fmt"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/helm"
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

			if preExecuteFunc, exists := d.PreExecuteFuncMap[step.Shared().ID]; exists {
				if err := preExecuteFunc(context.Background(), step); err != nil {
					level.Error(d.Logger).Log("event", "preExecute.fail", "err", err)
					return
				}
				// TODO(robert): need to store the progress for multiple occurrences of
				// a step with a pre execution func
				delete(d.PreExecuteFuncMap, step.ShortName())
			}

			d.hydrateAndSend(daemontypes.NewStep(step), c)
			return
		}
	}

	d.errNotFound(c)
}

func (d *NavcycleRoutes) hydrateStep(step daemontypes.Step) (*daemontypes.StepResponse, error) {
	debug := level.Debug(log.With(d.Logger, "method", "hydrateStep"))

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
		userValues := currentState.CurrentHelmValues()
		defaultValues := currentState.CurrentHelmValuesDefaults()
		releaseName := currentState.CurrentReleaseName()

		valuesFileContents, err := d.Fs.ReadFile(path.Join(constants.HelmChartPath, "values.yaml"))
		if err != nil {
			return nil, errors.Wrap(err, "read file values.yaml")
		}
		vendorValues := string(valuesFileContents)

		mergedValues, err := helm.MergeHelmValues(defaultValues, userValues, vendorValues, true)
		if err != nil {
			return nil, errors.Wrap(err, "merge values")
		}

		step.HelmValues.Values = mergedValues
		step.HelmValues.DefaultValues = vendorValues
		step.HelmValues.ReleaseName = releaseName
	}

	result := &daemontypes.StepResponse{
		CurrentStep: step,
		Phase:       step.Source.ShortName(),
	}

	debug.Log("event", "load.progress")
	if progress, ok := d.StepProgress.Load(step.Source.Shared().ID); ok {
		result.Progress = &progress
	}

	actions := d.getActions(result.CurrentStep)
	result.Actions = actions

	return result, nil
}

func (d *NavcycleRoutes) getActions(step daemontypes.Step) []daemontypes.Action {
	progress, hasProgress := d.StepProgress.Load(step.Source.Shared().ID)

	shouldSkipActions := hasProgress && progress.Status() != "success"

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
