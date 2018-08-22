package daemon

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func (d *NavcycleRoutes) terraformApply(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "terraformApply"))

	debug.Log("event", "confirm.terraformPlan")
	d.TerraformConfirmed <- true

	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "confirmed",
	})
}

func (d *NavcycleRoutes) terraformSkip(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "terraformSkip"))

	debug.Log("event", "deny.terraformPlan")
	d.TerraformConfirmed <- false

	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "denied",
	})
}
