package message

import (
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/spf13/viper"
)

type builderContext struct {
	logger log.Logger
	viper  *viper.Viper
	daemon daemontypes.Daemon
}

func (ctx builderContext) FuncMap() template.FuncMap {
	debug := level.Debug(log.With(ctx.logger, "step.type", "render", "render.phase", "template"))

	configFunc := func(name string) interface{} {
		configItemValue := ctx.viper.Get(name)
		if configItemValue == "" {
			debug.Log("event", "template.missing", "func", "config", "requested", name)
			return ""
		}
		return configItemValue
	}

	configItemFunc := func(name string) interface{} {
		if ctx.daemon == nil {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name)
			return ""
		}
		currentConfig, err := ctx.daemon.GetCurrentConfig()
		if err != nil {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name, "error", err)
			return ""
		}
		configItemValue, ok := currentConfig[name]
		if !ok {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name)
		}
		return configItemValue
	}

	return map[string]interface{}{
		"config":       configFunc,
		"ConfigOption": configItemFunc,
	}
}
