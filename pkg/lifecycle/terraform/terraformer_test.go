package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	mocktf "github.com/replicatedhq/ship/pkg/test-mocks/tfplan"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/version"
	"github.com/stretchr/testify/require"
)

func TestTerraformer(t *testing.T) {
	tests := []struct {
		name          string
		terraformFail bool
	}{
		{
			name: "zero",
		},
		{
			name:          "one",
			terraformFail: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := filepath.Join("/tmp", "ship-terraform", version.RunAtEpoch, "asset")
			if err := os.MkdirAll(d, 0755); err != nil {
				t.Fatal(err)
			}
			req := require.New(t)
			mc := gomock.NewController(t)
			mockDaemon := daemon.NewMockDaemon(mc)
			mockPlanner := mocktf.NewMockPlanConfirmer(mc)
			tf := &ForkTerraformer{
				Logger:        &logger.TestLogger{T: t},
				Daemon:        mockDaemon,
				PlanConfirmer: mockPlanner,
				Terraform: func() *exec.Cmd {
					cmd := exec.Command(os.Args[0], "-test.run=TestMockTerraform")
					cmd.Env = append(os.Environ(), "GOTEST_SUBPROCESS_MOCK=1")
					if test.terraformFail {
						cmd.Env = append(cmd.Env, "CRASHING_TERRAFORM_ERROR=1")
					}
					return cmd
				},
			}

			if test.terraformFail {
				err := tf.Execute(
					context.Background(),
					api.Release{},
					api.Terraform{},
				)
				req.Error(err)
				mc.Finish()
				return
			}

			mockPlanner.
				EXPECT().
				ConfirmPlan(gomock.Any(), "We're gonna make you some servers", gomock.Any()).
				Return(false, nil)

			err := tf.Execute(
				context.Background(),
				api.Release{},
				api.Terraform{},
			)
			req.NoError(err)
			mc.Finish()
		})
	}
}

func TestMockTerraform(t *testing.T) {
	if os.Getenv("GOTEST_SUBPROCESS_MOCK") == "" {
		return
	}

	if os.Getenv("CRASHING_TERRAFORM_ERROR") != "" {
		fmt.Fprintf(os.Stderr, os.Getenv("CRASHING_TERRAFORM_ERROR"))
		os.Exit(1)
	}

	receivedArgs := os.Args[2:]
	expectInit := []string{"init", "-input=false"}
	if reflect.DeepEqual(receivedArgs, expectInit) {
		os.Exit(0)
	}
}
