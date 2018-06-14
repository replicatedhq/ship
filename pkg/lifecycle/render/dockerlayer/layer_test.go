package dockerlayer

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/docker"
	"github.com/replicatedhq/ship/pkg/test-mocks/logger"
	"github.com/stretchr/testify/require"
)

func TestUnpackLayer(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "test layer",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			urlResolver := docker.NewMockPullURLResolver(mc)
			imageManager := docker.NewMockImageManager(mc)
			testLogger := &logger.TestLogger{T: t}
			ctx := context.Background()

			unpacker := &Unpacker{
				Logger:       testLogger,
				URLResolver:  urlResolver,
				ImageManager: imageManager,
			}

			asset := api.DockerLayerAsset{
				DockerAsset: api.DockerAsset{
					AssetShared: api.AssetShared{
						Dest: "some/where",
					},
					Image:  "replicated",
					Source: "public",
				},
				Layer: "abcdefg",
			}

			meta := api.ReleaseMetadata{
				Images: []api.Image{},
			}

			func() {
				defer mc.Finish()

				urlResolver.EXPECT().ResolvePullURL(&asset.DockerAsset, meta).Return("pull-url", nil)
				imageManager.EXPECT().ImagePull(ctx, "pull-url", types.ImagePullOptions{})

				err := unpacker.Execute(asset, meta)(ctx)
				req.NoError(err)
			}()
		})
	}
}
