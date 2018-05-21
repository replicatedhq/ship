package message

import (
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/api"
)

type builderContext struct {
	logger log.Logger
	viper  *viper.Viper
	daemon config.Daemon
	release *api.Release
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
		configItemValue, ok := ctx.daemon.GetCurrentConfig()[name]
		if !ok {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name)
		}
		return configItemValue
	}

	ctxFunc := func(name string) interface{} {
		switch name {
		case "state_file_path":
			return state.Path
		case "customer_id":
			return ctx.viper.GetString("customer-id")
		case "semver":
			return ctx.release.Metadata.Semver
		case "channel_name":
			return ctx.release.Metadata.ChannelName
		case "channel_id":
			return ctx.release.Metadata.ChannelID
		case "release_id":
			return ctx.release.Metadata.ReleaseID
		case "installation_id":
			return ctx.viper.GetString("installation-id")
		case "release_notes":
			return ctx.release.Metadata.ReleaseNotes
		}
		debug.Log("event", "template.missing", "func", "context", "requested", name)
		return ""
	}

	return map[string]interface{}{
		"config":       configFunc,
		"context":      ctxFunc, // TODO: this one's getting removed in 1.0
		"ConfigOption": configItemFunc,
		"Installation": ctxFunc,
	}
}
