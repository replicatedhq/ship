package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

type lifeycleStep struct {
	ID          string                `json:"id"`
	Description string                `json:"description"`
	Phase       string                `json:"phase"`
	Progress    *daemontypes.Progress `json:"progress,omitempty"`
}

func (d *V2Routes) getLifecycle(c *gin.Context) {
	lifecycleIDs := make([]lifeycleStep, 0)
	for _, step := range d.Release.Spec.Lifecycle.V1 {
		stepResponse := lifeycleStep{
			ID:          step.Shared().ID,
			Description: step.Shared().Description,
			Phase:       step.ShortName(),
		}

		if progress, ok := d.StepProgress.Load(step.Shared().ID); ok {
			stepResponse.Progress = &progress
		}

		lifecycleIDs = append(lifecycleIDs, stepResponse)
	}
	c.JSON(200, lifecycleIDs)
}
