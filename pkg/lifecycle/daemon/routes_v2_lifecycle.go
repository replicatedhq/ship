package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log/level"
)

type lifeycleStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Phase       string `json:"phase"`
}

func (d *V2Routes) getLifecycle(c *gin.Context) {
	lifecycleIDs := make([]lifeycleStep, 0)
	for _, step := range d.Release.Spec.Lifecycle.V1 {
		level.Debug(d.Logger).Log("step", step)

		lifecycleIDs = append(lifecycleIDs, lifeycleStep{
			ID:          step.Shared().ID,
			Description: step.Shared().Description,
			Phase:       step.ShortName(),
		})
	}
	c.JSON(200, lifecycleIDs)
}
