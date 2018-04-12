package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
)

const (
	ContainerImageSourceReplicated = "replicated"
	ContainerImageSourcePublic     = "public"
)

type DockerAssetBuilder struct {
	FS          afero.Afero
	ImagePuller ImagePuller
	ImageTags   ImageTags
}

// ImageTags defines functionality for resolving the image tag for an asset
type ImageTags interface {
	ResolveImageTag(ctx context.Context, asset api.DockerAsset) (string, error)
}

var _ ImageTags = &PassThruImageTags{}

// PassThruImageTags simply passes through the resolved asset
type PassThruImageTags struct{}

// ResolveImageTag by returning the given tag
func (b *PassThruImageTags) ResolveImageTag(ctx context.Context, asset api.DockerAsset) (string, error) {
	return asset.Image, nil
}

// ImagePuller defines the docker/docker client's ImagePull method
type ImagePuller interface {
	ImagePull(
		ctx context.Context,
		refStr string,
		options types.ImagePullOptions,
	) (io.ReadCloser, error)
}

// Execute pulls the image and tags it appropriately, then saves to the specified output dir
func (b *DockerAssetBuilder) Execute(ctx context.Context) error {
	//dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	return nil
}
