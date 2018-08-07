package config

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(context.Context, *api.Release, map[string]interface{}) (map[string]interface{}, error)
}

func NewResolver(
	logger log.Logger,
	daemon daemontypes.Daemon,
) Resolver {
	return &DaemonResolver{
		Logger: logger,
		Daemon: daemon,
	}

}
