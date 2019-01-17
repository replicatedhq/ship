package ship

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	"github.com/replicatedhq/ship/pkg/test-mocks/util"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name           string
		release        *api.Release
		selector       *replicatedapp.Selector
		uploadAssetsTo string
		expectError    error
	}{
		{
			name:    "execute",
			release: &api.Release{},
		},
		{
			name:           "execute with upload",
			release:        &api.Release{},
			uploadAssetsTo: "https://s3.amazonaws.com/some-bucket/some-key",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			d := daemon.NewMockDaemon(mc)
			uploader := util.NewMockAssetUploader(mc)
			testLogger := &logger.TestLogger{T: t}

			s := &Ship{
				Viper:          viper.New(),
				Headless:       false,
				Navcycle:       true,
				UploadAssetsTo: test.uploadAssetsTo,
				Logger:         testLogger,
				Daemon:         d,
				Uploader:       uploader,
			}

			ctx := context.Background()
			d.EXPECT().EnsureStarted(ctx, test.release)
			d.EXPECT().AwaitShutdown().Return(nil)
			uploader.EXPECT().UploadAssets(test.uploadAssetsTo)

			err := s.execute(ctx, test.release, test.selector)

			if test.expectError == nil {
				req.NoError(err)
			}
		})
	}
}
