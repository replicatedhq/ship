package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	uidaemon "github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	mocktf "github.com/replicatedhq/ship/pkg/test-mocks/tfplan"
	"github.com/replicatedhq/ship/pkg/testing/logger"
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
		apply             subproc
		expectConfirmPlan bool
		expectPlan        string
		expectApply       bool
		expectApplyOutput string
		expectError       bool
	}{
		{
			name: "init plan apply success",
			init: subproc{
				ExpectArgv: []string{"init", "-input=false"},
			},
			plan: subproc{
				ExpectArgv: []string{"plan", "-input=false", "-out=plan"},
				Stdout:     fmt.Sprintf("state%sCreating 1 cluster%show to apply", tfSep, tfSep),
			},
			apply: subproc{
				ExpectArgv: []string{"apply", "-input=false", "-auto-approve=true", "plan"},
				Stdout:     "Applied",
			},
			expectConfirmPlan: true,
			expectPlan:        `<div class="term-container">Creating 1 cluster</div>`,
			expectApply:       true,
			expectApplyOutput: `<div class="term-container">Applied</div>`,
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
			plan: subproc{
				Stdout: fmt.Sprintf("\n\n%s\n\n", tfNoChanges),
			},
		},
		{
			name: "apply fail",
			plan: subproc{
				Stdout: fmt.Sprintf("state%sCreating 1 cluster%show to apply", tfSep, tfSep),
			},
			expectConfirmPlan: true,
			expectPlan:        `<div class="term-container">Creating 1 cluster</div>`,
			apply: subproc{
				Fail: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockDaemon := daemon.NewMockDaemon(mc)
			mockPlanner := mocktf.NewMockPlanConfirmer(mc)
			tf := &ForkTerraformer{
				Logger:        &logger.TestLogger{T: t},
				Daemon:        mockDaemon,
				PlanConfirmer: mockPlanner,
				Terraform: func(string) *exec.Cmd {
					cmd := exec.Command(os.Args[0], "-test.run=TestMockTerraform")

					init, err := json.Marshal(test.init)
					if err != nil {
						log.Fatal(err)
					}

					plan, err := json.Marshal(test.plan)
					if err != nil {
						log.Fatal(err)
					}

					apply, err := json.Marshal(test.apply)
					if err != nil {
						log.Fatal(err)
					}

					cmd.Env = append(os.Environ(),
						"GOTEST_SUBPROCESS_MOCK=1",
						"TERRAFORM_INIT="+string(init),
						"TERRAFORM_PLAN="+string(plan),
						"TERRAFORM_APPLY="+string(apply),
					)
					return cmd
				},
			}

			if test.expectConfirmPlan {
				mockPlanner.
					EXPECT().
					ConfirmPlan(gomock.Any(), test.expectPlan, gomock.Any(), gomock.Any()).
					Return(test.expectApply, nil)
			}

			if test.expectApply {
				mockDaemon.
					EXPECT().
					PushStreamStep(gomock.Any(), gomock.Any(), gomock.Any())

				msg := uidaemon.Message{
					Contents:    test.expectApplyOutput,
					TrustedHTML: true,
				}
				if test.apply.Fail {
					msg.Level = "error"
				}

				mockDaemon.
					EXPECT().
					PushMessageStep(gomock.Any(), msg, gomock.Any(), gomock.Any()).
					Return()

				if test.apply.Fail {
					ch := make(chan bool, 1)
					ch <- false
					mockDaemon.
						EXPECT().
						TerraformConfirmedChan().
						Return(ch)
				} else {
					ch := make(chan string, 1)
					ch <- ""
					mockDaemon.
						EXPECT().
						MessageConfirmedChan().
						Return(ch)
				}
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
	case "apply":
		env = "TERRAFORM_APPLY"
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

func TestForkTerraformerApply(t *testing.T) {
	req := require.New(t)
	mc := gomock.NewController(t)
	mockDaemon := daemon.NewMockDaemon(mc)
	ft := ForkTerraformer{
		Daemon: mockDaemon,
		Logger: &logger.TestLogger{T: t},
		Terraform: func(string) *exec.Cmd {
			cmd := exec.Command(os.Args[0], "-test.run=TestMockTerraformApply")
			cmd.Env = append(os.Environ(),
				"GOTEST_SUBPROCESS_MOCK=1",
			)
			return cmd
		},
	}

	msgs := make(chan uidaemon.Message, 10)
	output, err := ft.apply(msgs)
	req.NoError(err)
	req.Equal(output, `<div class="term-container">stdout1stderr1stdout2</div>`)

	req.EqualValues(uidaemon.Message{
		Contents:    `<div class="term-container">terraform apply</div>`,
		TrustedHTML: true,
	}, <-msgs)

	req.EqualValues(uidaemon.Message{
		Contents:    `<div class="term-container">stdout1</div>`,
		TrustedHTML: true,
	}, <-msgs)

	req.EqualValues(uidaemon.Message{
		Contents:    `<div class="term-container">stdout1stderr1</div>`,
		TrustedHTML: true,
	}, <-msgs)

	req.EqualValues(uidaemon.Message{
		Contents:    `<div class="term-container">stdout1stderr1stdout2</div>`,
		TrustedHTML: true,
	}, <-msgs)
}

func TestMockTerraformApply(t *testing.T) {
	if os.Getenv("GOTEST_SUBPROCESS_MOCK") == "" {
		return
	}

	fmt.Fprintf(os.Stdout, "stdout1")
	time.Sleep(time.Millisecond)
	fmt.Fprintf(os.Stderr, "stderr1")
	time.Sleep(time.Millisecond)
	fmt.Fprintf(os.Stdout, "stdout2")

	os.Exit(0)
}
