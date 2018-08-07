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

type NoOpResolver struct {
}

func (NoOpResolver) ResolveConfig(context.Context, *api.Release, map[string]interface{}) (map[string]interface{}, error) {
	// todo load from state or something
	return map[string]interface{}{}, nil
}

func NewNoOpResolver() Resolver {
	return &NoOpResolver{}
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
