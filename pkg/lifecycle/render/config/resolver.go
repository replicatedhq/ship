package config

import (
	"context"
)

// IResolver is a thing that can resolve configuration options
type IResolver interface {
	ResolveConfig(ctx context.Context) (map[string]interface{}, error)
}
