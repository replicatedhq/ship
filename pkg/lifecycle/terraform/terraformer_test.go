package terraform

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	mocktf "github.com/replicatedhq/ship/pkg/test-mocks/tfplan"
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
			mockDaemon := daemon.NewMockDaemon(mc)
			mockPlanner := mocktf.NewMockPlanConfirmer(mc)
			tf := &ForkTerraformer{
				Logger:        &logger.TestLogger{T: t},
				Daemon:        mockDaemon,
				PlanConfirmer: mockPlanner,
			}

			mockPlanner.
				EXPECT().
				ConfirmPlan(gomock.Any(), "We're gonna make you some servers", gomock.Any()).
				Return(false, nil)

			//err := tf.Execute(
			tf.Execute(
				context.Background(),
				api.Release{},
				api.Terraform{},
			)
			//req.NoError(err)
			mc.Finish()
		})
	}
}
