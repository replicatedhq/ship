package replicatedapp

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPersistSpec(t *testing.T) {

	r := &resolver{
		FS: afero.Afero{Fs: afero.NewMemMapFs()},
	}

	req := require.New(t)

	desiredSpec := []byte("my cool spec")
	err := r.persistSpec(desiredSpec)
	req.NoError(err)

	persistedSpec, err := r.FS.ReadFile(constants.ReleasePath)
	req.NoError(err)
	req.True(reflect.DeepEqual(desiredSpec, persistedSpec))
}

func TestPersistRelease(t *testing.T) {
	tests := []struct {
		name          string
		inputRelease  *ShipRelease
		inputSelector *Selector
		shaSummer     shaSummer
		expectCalls   func(t *testing.T, stateManager *state.MockManager)
		expectRelease *api.Release
	}{
		{
			name: "happy path",
			inputRelease: &ShipRelease{
				ID: "12345",
				Spec: `
---
assets: 
  v1: []
`,
			},
			inputSelector: &Selector{
				CustomerID:     "kfbr",
				InstallationID: "392",
			},
			shaSummer: func(bytes []byte) string {
				return "abcdef"
			},
			expectCalls: func(t *testing.T, stateManager *state.MockManager) {
				stateManager.EXPECT().SerializeAppMetadata(&matchers.Is{
					Test: func(v interface{}) bool {
						rm := v.(api.ReleaseMetadata)
						t.Log("testing release metadata", fmt.Sprintf("%v", rm))
						return rm.ReleaseID == "12345" &&
							rm.CustomerID == "kfbr" &&
							rm.InstallationID == "392"
					},
				})
				stateManager.EXPECT().SerializeContentSHA("abcdef")
			},
			expectRelease: &api.Release{
				Spec: api.Spec{
					Assets: api.Assets{
						V1: []api.Asset{},
					},
				},
				Metadata: api.ReleaseMetadata{
					ReleaseID:      "12345",
					CustomerID:     "kfbr",
					InstallationID: "392",
					Images:         []api.Image{},
					GithubContents: []api.GithubContent{},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			stateManager := state.NewMockManager(mc)
			defer mc.Finish()

			test.expectCalls(t, stateManager)

			resolver := &resolver{
				Logger:       &logger.TestLogger{T: t},
				StateManager: stateManager,
				ShaSummer:    test.shaSummer,
			}

			result, err := resolver.persistRelease(test.inputRelease, test.inputSelector)

			req.NoError(err)
			req.Equal(test.expectRelease, result)
		})
	}
}
