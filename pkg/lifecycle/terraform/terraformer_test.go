package terraform

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/config"
	"github.com/replicatedhq/ship/pkg/testing/logger"
)

func TestTerraformer(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "zero",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			//req := require.New(t)
			mc := gomock.NewController(t)
			mockDaemon := config.NewMockDaemon(mc)
			tf := &ForkTerraformer{
				Logger: &logger.TestLogger{T: t},
				Daemon: mockDaemon,
			}
			//err := tf.Execute(
			tf.Execute(
				context.Background(),
				api.Release{},
				api.Terraform{},
			)
			//req.NoError(err)
		})
	}
}
