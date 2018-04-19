package config

import (
	"context"

	"github.com/replicatedcom/ship/pkg/api"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(*api.ReleaseMetadata, context.Context) (map[string]interface{}, error)
}
