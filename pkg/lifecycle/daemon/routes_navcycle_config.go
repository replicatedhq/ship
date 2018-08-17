package daemon

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
)

func (d *NavcycleRoutes) postAppConfigLive(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "postAppConfigLive"))

		type Request struct {
			ItemValues []ConfigOption `json:"item_values"`
		}

		debug.Log("event", "request.bind")
		var request Request
		if err := c.BindJSON(&request); err != nil {
			level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
			return
		}

		debug.Log("event", "state.tryLoad")
		savedSate, err := d.StateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		liveValues := make(map[string]interface{})
		for _, itemValue := range request.ItemValues {
			liveValues[itemValue.Name] = itemValue.Value
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := d.ConfigRenderer.ResolveConfig(c, release, savedSate.CurrentConfig(), liveValues, true)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		type Result struct {
			Version int
			Groups  []libyaml.ConfigGroup
		}
		r := Result{
			Version: 1,
			Groups:  resolvedConfig,
		}

		debug.Log("event", "returnLiveConfig")
		c.JSON(200, r)
	}
}

func (d *NavcycleRoutes) finalizeAppConfig(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "finalizeAppConfig"))
		d.putAppConfig(release)(c)
		debug.Log("event", "configSaved.send.start")
		d.ConfigSaved <- nil
		debug.Log("event", "configSaved.send.complete")
	}
}

func (d *NavcycleRoutes) putAppConfig(release *api.Release) gin.HandlerFunc {
	return func(c *gin.Context) {
		debug := level.Debug(log.With(d.Logger, "handler", "putAppConfig"))

		type Request struct {
			Options  []ConfigOption `json:"options"`
			Validate bool           `json:"validate"`
		}

		debug.Log("event", "request.bind")
		var request Request
		if err := c.BindJSON(&request); err != nil {
			level.Error(d.Logger).Log("event", "unmarshal request failed", "err", err)
			return
		}

		debug.Log("event", "state.tryLoad")
		savedState, err := d.StateManager.TryLoad()
		if err != nil {
			level.Error(d.Logger).Log("msg", "failed to load stateManager", "err", err)
			c.AbortWithStatus(500)
			return
		}

		liveValues := make(map[string]interface{})
		for _, itemValue := range request.Options {
			liveValues[itemValue.Name] = itemValue.Value
		}

		debug.Log("event", "resolveConfig")
		resolvedConfig, err := d.ConfigRenderer.ResolveConfig(c, release, savedState.CurrentConfig(), liveValues, false)
		if err != nil {
			level.Error(d.Logger).Log("event", "resolveconfig failed", "err", err)
			c.AbortWithStatus(500)
			return
		}

		if validationErrors := resolve.ValidateConfig(resolvedConfig); validationErrors != nil {
			c.AbortWithStatusJSON(400, validationErrors)
			return
		}

		// NOTE: what about multi value, data, multi data?
		templateContext := make(map[string]interface{})
		for _, configGroup := range resolvedConfig {
			for _, configItem := range configGroup.Items {
				templateContext[configItem.Name] = configItem.Value
			}
		}

		debug.Log("event", "state.serialize")
		if err := d.StateManager.SerializeConfig(nil, api.ReleaseMetadata{}, templateContext); err != nil {
			level.Error(d.Logger).Log("msg", "serialize state failed", "err", err)
			c.AbortWithStatus(500)
		}

		d.CurrentConfig = templateContext
		c.JSON(200, make(map[string]interface{}))
	}
}
