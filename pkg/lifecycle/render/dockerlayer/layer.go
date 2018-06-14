package dockerlayer

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
)

type Unpacker struct {
	Logger       log.Logger
	URLResolver  docker.PullURLResolver
	ImageManager docker.ImageManager
}

func NewUnpacker(
	logger log.Logger,
	resolver docker.PullURLResolver,
	manager docker.ImageManager,
) *Unpacker {
	return &Unpacker{
		Logger:       logger,
		URLResolver:  resolver,
		ImageManager: manager,
	}
}

func (u *Unpacker) Execute(
	asset api.DockerLayerAsset,
	meta api.ReleaseMetadata,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return nil
	}
}
