package config

import (
	"context"

	"github.com/replicatedcom/ship/pkg/api"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(context.Context, *api.ReleaseMetadata, map[string]interface{}) (map[string]interface{}, error)
}
