package config

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(context.Context, *api.Release, map[string]interface{}) (map[string]interface{}, error)
	WithDaemon(d daemon.Daemon) Resolver
}

func NewResolver(logger log.Logger) Resolver {
	return &DaemonResolver{
		Logger: logger,
	}

}
func (r *DaemonResolver) WithDaemon(d daemon.Daemon) Resolver {
	r.Daemon = d
	return r
}

func NewDaemon(
	v *viper.Viper,
	headless *daemon.HeadlessDaemon,
	headed *daemon.ShipDaemon,
) daemon.Daemon {
	if v.GetBool("headless") {
		return headless
	}
	return headed
}

func NewHeadlessDaemon(
	v *viper.Viper,
	logger log.Logger,
	renderer *resolve.APIConfigRenderer,
	stateManager *state.Manager,
) *daemon.HeadlessDaemon {
	return &daemon.HeadlessDaemon{
		StateManager:   stateManager,
		Logger:         logger,
		UI:             ui.FromViper(v),
		ConfigRenderer: renderer,
	}
}

func NewHeadedDaemon(
	v *viper.Viper,
	renderer *resolve.APIConfigRenderer,
	stateManager *state.Manager,
	logger log.Logger,
	ui cli.Ui,
	fs afero.Afero,
) *daemon.ShipDaemon {
	return &daemon.ShipDaemon{
		Logger:           logger,
		Fs:               fs,
		UI:               ui,
		StateManager:     stateManager,
		Viper:            v,
		ConfigSaved:      make(chan interface{}),
		MessageConfirmed: make(chan string, 1),
		ConfigRenderer:   renderer,
	}

}
