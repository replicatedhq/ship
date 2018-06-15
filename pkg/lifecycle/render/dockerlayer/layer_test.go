package dockerlayer

import (
	"context"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	mockdocker "github.com/replicatedhq/ship/pkg/test-mocks/docker"
	"github.com/replicatedhq/ship/pkg/test-mocks/logger"
	"github.com/spf13/viper"
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
			renderer := mockdocker.NewMockRenderer(mc)
			archiver := NewMockA
			testLogger := &logger.TestLogger{T: t}
			ctx := context.Background()

			unpacker := &Unpacker{
				Logger:      testLogger,
				Viper:       viper.New(),
				DockerSaver: renderer,
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

			watchProgress := func(ch chan interface{}, logger log.Logger) error {
				return nil
			}

			func() {
				defer mc.Finish()

				renderer.EXPECT().Execute(asset.DockerAsset, meta, watchProgress).Return(func(ctx2 context.Context) error {
					// todo make sure this thing got called
					return nil
				})

				err := unpacker.Execute(asset, meta, watchProgress)(ctx)
				req.NoError(err)
			}()
		})
	}
}
