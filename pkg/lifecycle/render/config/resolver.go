package config

import (
	"context"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/fs"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/spf13/viper"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(context.Context, *api.Release, map[string]interface{}) (map[string]interface{}, error)
	WithDaemon(d Daemon) Resolver
}

func ResolverFromViper(v *viper.Viper) Resolver {
	return &DaemonResolver{
		Logger: logger.FromViper(v),
	}

}
func (r *DaemonResolver) WithDaemon(d Daemon) Resolver {
	r.Daemon = d
	return r
}

func DaemonFromViper(v *viper.Viper) Daemon {
	if v.GetBool("headless") {
		return &HeadlessDaemon{
			StateManager: state.ManagerFromViper(v),
			Logger:       logger.FromViper(v),
			UI:           ui.FromViper(v),
		}
	}

	renderer := &APIConfigRenderer{
		Logger:         logger.FromViper(v),
		Viper:          v,
		BuilderBuilder: templates.BuilderBuilderFromViper(v),
	}

	return &ShipDaemon{
		Logger:           logger.FromViper(v),
		Fs:               fs.FromViper(v),
		UI:               ui.FromViper(v),
		StateManager:     state.ManagerFromViper(v),
		Viper:            v,
		ConfigSaved:      make(chan interface{}),
		MessageConfirmed: make(chan string, 1),
		ConfigRenderer:   renderer,
	}
}
