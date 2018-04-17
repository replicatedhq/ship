package config

import (
	"context"
)

// Resolver is a thing that can resolve configuration options
type Resolver interface {
	ResolveConfig(ctx context.Context) (map[string]interface{}, error)
}
