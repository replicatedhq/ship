package terraform

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/terraform"
	"github.com/replicatedhq/ship/pkg/state"
	state2 "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPersistState(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		instate  state.State
		outstate state.State
	}{
		{
			name: "post-delete state, mostly empty",
			state: `{
    "version": 3,
    "terraform_version": "0.11.9",
    "serial": 3,
    "lineage": "3218b0ca-45de-2227-e420-f0d948f91e86",
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {},
            "resources": {},
            "depends_on": []
        }
    ]
}
`,
			instate: state.State{
				V1: &state.V1{},
			},
			outstate: state.State{
				V1: &state.V1{
					Terraform: &state.Terraform{
						RawState: `{
    "version": 3,
    "terraform_version": "0.11.9",
    "serial": 3,
    "lineage": "3218b0ca-45de-2227-e420-f0d948f91e86",
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {},
            "resources": {},
            "depends_on": []
        }
    ]
}
`,
						State: &terraform.State{
							Version:   3,
							TFVersion: "0.11.9",
							Serial:    3,
							Lineage:   "3218b0ca-45de-2227-e420-f0d948f91e86",
							Modules: []*terraform.ModuleState{
								{
									Path: []string{
										"root",
									},
									Outputs:      map[string]*terraform.OutputState{},
									Resources:    map[string]*terraform.ResourceState{},
									Dependencies: []string{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			req := require.New(t)
			debug := &logger.TestLogger{T: t}
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			statemanager := state2.NewMockManager(mc)

			err := mockFs.WriteFile("installer/terraform.tfstate", []byte(test.state), 0644)
			req.NoError(err)

			statemanager.EXPECT().CachedState().Return(test.instate, nil)
			statemanager.EXPECT().Save(&matchers.Is{
				Test: func(v interface{}) bool {
					vstate := v.(state.State)
					diff := deep.Equal(*vstate.V1.Terraform, *test.outstate.V1.Terraform)
					t.Log(strings.Join(diff, "\n"))
					return len(diff) == 0
				},
				Describe: fmt.Sprintf("equal to %+v", *test.outstate.V1.Terraform),
			}).Return(nil)

			err = persistState(debug, mockFs, statemanager, "installer")
			req.NoError(err)

			mc.Finish()

		})
	}
}

func TestRestoreState(t *testing.T) {
	tests := []struct {
		name       string
		instate    state.State
		expectFile string
	}{
		{
			name: "post-delete state, mostly empty",
			expectFile: `{
    "version": 3,
    "terraform_version": "0.11.9",
    "serial": 3,
    "lineage": "3218b0ca-45de-2227-e420-f0d948f91e86",
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {},
            "resources": {},
            "depends_on": []
        }
    ]
}
`,
			instate: state.State{
				V1: &state.V1{
					Terraform: &state.Terraform{
						RawState: `{
    "version": 3,
    "terraform_version": "0.11.9",
    "serial": 3,
    "lineage": "3218b0ca-45de-2227-e420-f0d948f91e86",
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {},
            "resources": {},
            "depends_on": []
        }
    ]
}
`,
						State: &terraform.State{
							Version:   3,
							TFVersion: "0.11.9",
							Serial:    3,
							Lineage:   "3218b0ca-45de-2227-e420-f0d948f91e86",
							Modules: []*terraform.ModuleState{
								{
									Path: []string{
										"root",
									},
									Outputs:      map[string]*terraform.OutputState{},
									Resources:    map[string]*terraform.ResourceState{},
									Dependencies: []string{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			req := require.New(t)
			debug := &logger.TestLogger{T: t}
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			statemanager := state2.NewMockManager(mc)

			statemanager.EXPECT().CachedState().Return(test.instate, nil)

			err := restoreState(debug, mockFs, statemanager, "installer")
			req.NoError(err)

			contents, err := mockFs.ReadFile(path.Join("installer", "terraform.tfstate"))
			req.NoError(err)

			req.Equal(test.expectFile, string(contents))

			mc.Finish()

		})
	}
}
