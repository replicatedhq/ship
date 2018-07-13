package config

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
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
