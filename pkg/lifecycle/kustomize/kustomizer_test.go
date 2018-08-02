package kustomize

import (
	"context"
	"testing"

	"path"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/state"
	daemon2 "github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	state2 "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestKustomizer(t *testing.T) {
	tests := []struct {
		name        string
		kustomize   state.Kustomize
		expectFiles map[string]string
	}{
		{
			name: "no files",
			kustomize: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Patches:           make(map[string]string),
						KustomizationYAML: "",
					},
				},
			},
			expectFiles: map[string]string{},
		},
		{
			name: "one file",
			kustomize: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Patches: map[string]string{
							"deployment.yml": `---
metadata:
  name: my-deploy
spec:
  replicas: 100`,
						},
						KustomizationYAML: "",
					},
				},
			},
			expectFiles: map[string]string{
				"deployment.yml": `---
metadata:
  name: my-deploy
spec:
  replicas: 100`,

				"kustomization.yml": `patches:
- deployment.yml
`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			testLogger := &logger.TestLogger{T: t}
			mockDaemon := daemon2.NewMockDaemon(mc)
			mockState := state2.NewMockManager(mc)

			saveChan := make(chan interface{})
			close(saveChan)

			ctx := context.Background()
			release := api.Release{}

			mockDaemon.EXPECT().EnsureStarted(ctx, &release)
			mockDaemon.EXPECT().PushKustomizeStep(ctx, daemon.Kustomize{
				BasePath: "someBasePath",
			})
			mockDaemon.EXPECT().KustomizeSavedChan().Return(saveChan)
			mockState.EXPECT().TryLoad().Return(state.VersionedState{V1: &state.V1{
				Kustomize: &test.kustomize,
			}}, nil)

			k := &kustomizer{
				Logger: testLogger,
				FS:     mockFS,
				Daemon: mockDaemon,
				State:  mockState,
			}

			err := k.Execute(
				ctx,
				release,
				api.Kustomize{
					BasePath: "someBasePath",
					Dest:     "overlays/ship",
				},
			)

			for name, contents := range test.expectFiles {
				pathToFile := path.Join("overlays", "ship", name)
				actual, err := mockFS.ReadFile(pathToFile)
				req.NoError(err, "read expected file %s", pathToFile)
				req.Equal(contents, string(actual))
			}

			req.NoError(err)
		})
	}
}
