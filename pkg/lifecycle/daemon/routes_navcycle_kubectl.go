package daemon

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func (d *NavcycleRoutes) kubectlConfirm(c *gin.Context) {
	debug := level.Debug(log.With(d.Logger, "method", "kubectlConfirm"))

	debug.Log("event", "confirm.kubectl")
	d.KubectlConfirmed <- true

	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "confirmed",
	})
}
