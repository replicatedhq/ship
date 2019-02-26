package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/ship/pkg/api"
)

func (d *NavcycleRoutes) getMetadata(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch release.Metadata.Type {
		case "helm":
			fallthrough
		case "k8s":
			c.JSON(200, release.Metadata.ShipAppMetadata)
			return
		case "runbook.replicated.app":
			fallthrough
		case "replicated.app":
			fallthrough
		case "inline.replicated.app":
			c.JSON(200, map[string]interface{}{
				"name": release.Metadata.ChannelName,
				"icon": release.Metadata.ChannelIcon,
			})
			return
		}
	}
}
