package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

type subproc struct {
	Fail       bool
	Stdout     string
	Stderr     string
	ExpectArgv []string
}

func TestTerraformer(t *testing.T) {
	tests := []struct {
		name              string
		init              subproc
		plan              subproc
		expectConfirmPlan bool
		expectPlan        string
		expectError       bool
	}{
		{
			name: "init plan success",
			init: subproc{
				ExpectArgv: []string{"init", "-input=false"},
			},
			plan: subproc{
				Stdout: fmt.Sprintf("state%sCreating 1 cluster%show to apply", tfSep, tfSep),
			},
			expectConfirmPlan: true,
			expectPlan:        `<div class="term-container">Creating 1 cluster</div>`,
		},
		{
			name: "init fail",
			init: subproc{
				Fail: true,
			},
			expectError: true,
		},
		{
			name: "plan no changes",
			init: subproc{
				ExpectArgv: []string{"init", "-input=false"},
			},
			plan: subproc{
				Stdout: fmt.Sprintf("\n\n%s\n\n", tfNoChanges),
			},
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

					init, err := json.Marshal(test.init)
					if err != nil {
						log.Fatal(err)
					}

					plan, err := json.Marshal(test.plan)
					if err != nil {
						log.Fatal(err)
					}

					cmd.Env = append(os.Environ(),
						"GOTEST_SUBPROCESS_MOCK=1",
						"TERRAFORM_INIT="+string(init),
						"TERRAFORM_PLAN="+string(plan),
					)
					return cmd
				},
			}

			if test.expectConfirmPlan {
				mockPlanner.
					EXPECT().
					ConfirmPlan(gomock.Any(), test.expectPlan, gomock.Any()).
					Return(false, nil)
			}

			err := tf.Execute(
				context.Background(),
				api.Release{},
				api.Terraform{},
			)

			if test.expectError {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			mc.Finish()
		})
	}
}

func TestMockTerraform(t *testing.T) {
	if os.Getenv("GOTEST_SUBPROCESS_MOCK") == "" {
		return
	}

	var env string
	switch os.Args[2] {
	case "init":
		env = "TERRAFORM_INIT"
	case "plan":
		env = "TERRAFORM_PLAN"
	}

	var config subproc
	err := json.Unmarshal([]byte(os.Getenv(env)), &config)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.ExpectArgv) > 0 {
		receivedArgs := os.Args[2:]
		if !reflect.DeepEqual(receivedArgs, config.ExpectArgv) {
			fmt.Fprintf(os.Stderr, "; FAIL expected args %+v got %+v", config.ExpectArgv, receivedArgs)
			os.Exit(2)
		}
	}

	fmt.Fprintf(os.Stdout, config.Stdout)
	fmt.Fprintf(os.Stderr, config.Stderr)

	if config.Fail {
		os.Exit(1)
	}

	os.Exit(0)
}
