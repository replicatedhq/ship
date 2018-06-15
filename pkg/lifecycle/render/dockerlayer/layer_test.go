package dockerlayer

import (
	"context"
	"testing"

	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	mockdocker "github.com/replicatedhq/ship/pkg/test-mocks/docker"
	"github.com/replicatedhq/ship/pkg/test-mocks/dockerlayer"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestUnpackLayer(t *testing.T) {
	tests := []struct {
		name        string
		dockerError error
	}{
		{
			name:        "test layer",
			dockerError: nil,
		},
		{
			name:        "test layer with docker save error",
			dockerError: errors.New("image not found"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			renderer := mockdocker.NewMockRenderer(mc)
			archiver := dockerlayer.NewMockArchiver(mc)
			testLogger := &logger.TestLogger{T: t}
			ctx := context.Background()
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}

			unpacker := &Unpacker{
				Logger:      testLogger,
				Viper:       viper.New(),
				DockerSaver: renderer,
				Tar:         archiver,
				FS:          mockFS,
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

				var calledDockerExec bool
				renderer.EXPECT().Execute(asset.DockerAsset, meta, gomock.Any(), gomock.Any()).Return(func(ctx2 context.Context) error {
					// todo make sure this thing got called
					calledDockerExec = true
					return test.dockerError
				})

				if test.dockerError == nil {
					archiver.EXPECT().Open(&startswith{"/tmp/dockerlayer"}, &startswith{"/tmp/dockerlayer"}).Return(nil)
					archiver.EXPECT().Open(&startswith{"/tmp/dockerlayer"}, asset.Dest).Return(nil)
				}

				err := unpacker.Execute(asset, meta, watchProgress)(ctx)

				if test.dockerError != nil {
					req.Error(err, "expected error "+test.dockerError.Error())
					req.Contains(err.Error(), test.dockerError.Error())
					return
				}

				req.NoError(err)
				dirCreated, err := mockFS.Exists("some/")
				req.NoError(err, "check dir created")
				req.True(dirCreated, "expected base dir to have been created at %s", "some/")

				req.True(calledDockerExec, "expected docker.Renderer.Execute to have been called")
			}()
		})
	}
}

var _ gomock.Matcher = &startswith{}

type startswith struct {
	value string
}

func (s *startswith) String() string {
	return fmt.Sprintf("start with %s", s.value)
}
func (s *startswith) Matches(x interface{}) bool {
	str := fmt.Sprintf("%v", x)
	return strings.HasPrefix(str, s.value)
}
