package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
)

type V2Routes struct {
	Logger log.Logger
}

func (d *V2Routes) Register(group *gin.RouterGroup, release *api.Release) {
	v2 := group.Group("/api/v2")
	v2.GET("/lifecycle", d.getLifecycle(release))
}

func (d *V2Routes) getLifecycle(release *api.Release) gin.HandlerFunc {
	type DaemonStep struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Phase       string `json:"phase"`
	}
	return func(c *gin.Context) {
		var lifecycleIDs []DaemonStep
		for _, step := range release.Spec.Lifecycle.V1 {
			level.Debug(d.Logger).Log("step", step)

			lifecycleIDs = append(lifecycleIDs, DaemonStep{
				// TODO pull these from branch
				//ID:          step.Shared().ID,
				//Description: step.Shared().Description,
				//Phase:       step.ShortName(),
			})
		}
		c.JSON(200, lifecycleIDs)
	}
}
