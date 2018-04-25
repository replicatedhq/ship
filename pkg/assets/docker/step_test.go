package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/test-fixtures/docker"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	name   string
	asset  api.DockerAsset
	expect func(*docker.MockImagePuller)
}

func TestDockerAsset(t *testing.T) {
	req := require.New(t)
	tests := []testcase{
		{
			name: "the first one",
			asset: api.DockerAsset{
				AssetShared: api.AssetShared{
					Dest:        "docker/x5.tar",
					Mode:        0666,
					Description: "The x5 image",
				},
				Image:   "registry.replicated.com/team/i-know-what-the-x5-is:kfbr392",
				Private: true,
			},
			expect: func(mock *docker.MockImagePuller) {
				mock.EXPECT().ImagePull(
					gomock.Any(),
					"registry.replicated.com/team/FAKE_SLUG_KEY.i-know-what-the-x5-is:kfbr392",
					types.ImagePullOptions{
						RegistryAuth: "", // todo fetch with re-written key
					},
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			puller := docker.NewMockImagePuller(controller)
			puller.EXPECT()
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}

			builder := &DockerAssetBuilder{
				FS: mockFS,
			}

			err := builder.Execute(context.Background())

			req.NoError(err)
		})
	}
}
