package daemon

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
	mockstate "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/stretchr/testify/require"
)

type kustomizeSaveFileTestCase struct {
	Name        string
	InState     state.V1
	Body        SaveOverlayRequest
	ExpectState state.Kustomize
}

func TestV2KustomizeSaveFile(t *testing.T) {
	tests := []kustomizeSaveFileTestCase{
		{
			Name: "empty add patch",
			Body: SaveOverlayRequest{
				Contents: "foo/bar/baz",
				Path:     "deployment.yaml",
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						ExcludedBases: []string{},
						Patches: map[string]string{
							"deployment.yaml": "foo/bar/baz",
						},
						Resources: map[string]string{},
					},
				},
			},
		},
		{
			Name: "empty add resource",
			Body: SaveOverlayRequest{
				Contents:   "foo/bar/baz",
				Path:       "deployment.yaml",
				IsResource: true,
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						ExcludedBases: []string{},
						Patches:       map[string]string{},
						Resources: map[string]string{
							"deployment.yaml": "foo/bar/baz",
						},
					},
				},
			},
		},
		{
			Name: "add resource when patch exists",
			Body: SaveOverlayRequest{
				Contents:   "foo/bar/baz",
				Path:       "service.yaml",
				IsResource: true,
			},
			InState: state.V1{
				Kustomize: &state.Kustomize{
					Overlays: map[string]state.Overlay{
						"ship": {
							ExcludedBases: []string{},
							Patches: map[string]string{
								"deployment.yaml": "foo/bar/baz",
							},
						},
					},
				},
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						ExcludedBases: []string{},
						Patches: map[string]string{
							"deployment.yaml": "foo/bar/baz",
						},
						Resources: map[string]string{
							"service.yaml": "foo/bar/baz",
						},
					},
				},
			},
		},
		{
			Name: "merge into existing",
			Body: SaveOverlayRequest{
				Contents:   "foo/bar/baz",
				Path:       "service.yaml",
				IsResource: true,
			},
			InState: state.V1{
				Kustomize: &state.Kustomize{
					Overlays: map[string]state.Overlay{
						"ship": {
							ExcludedBases: []string{},
							Resources: map[string]string{
								"deployment.yaml": "foo/bar/baz",
							},
							Patches: map[string]string{
								"deployment.yaml": "foo/bar/baz",
							},
						},
					},
				},
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						ExcludedBases: []string{},
						Resources: map[string]string{
							"deployment.yaml": "foo/bar/baz",
							"service.yaml":    "foo/bar/baz",
						},
						Patches: map[string]string{
							"deployment.yaml": "foo/bar/baz",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			mc := gomock.NewController(t)
			fakeState := mockstate.NewMockManager(mc)
			testLogger := &logger.TestLogger{T: t}
			progressmap := &daemontypes.ProgressMap{}

			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StateManager: fakeState,
				StepProgress: progressmap,
			}

			fakeState.EXPECT().CachedState().Return(state.State{
				V1: &test.InState,
			}, nil).AnyTimes()

			fakeState.EXPECT().SaveKustomize(&matchers.Is{
				Test: func(v interface{}) bool {
					c, ok := v.(*state.Kustomize)
					if !ok {
						return false
					}
					diff := deep.Equal(&test.ExpectState, c)
					if len(diff) != 0 {
						fmt.Print(fmt.Sprintf("Failed diff compare with %s", strings.Join(diff, "\n")))
						return false
					}
					return true
				},
				Describe: fmt.Sprintf("expect state equal to %s", test.ExpectState),
			}).Return(nil).AnyTimes()

			err := v2.kustomizeDoSaveOverlay(test.Body)
			req.NoError(err)
			mc.Finish()
		})
	}
}

type kustomizeDeleteFileTestCase struct {
	Name             string
	InState          state.V1
	DeleteFileParams deleteFileParams
	ExpectState      state.Kustomize
}

type deleteFileParams struct {
	pathQueryParam string
	getFiles       func(overlay state.Overlay) map[string]string
}

func TestV2KustomizeDeleteFile(t *testing.T) {
	tests := []kustomizeDeleteFileTestCase{
		{
			Name: "delete patch",
			DeleteFileParams: deleteFileParams{
				pathQueryParam: "deployment.yaml",
				getFiles: func(overlay state.Overlay) map[string]string {
					return overlay.Patches
				},
			},
			InState: state.V1{
				Kustomize: &state.Kustomize{
					Overlays: map[string]state.Overlay{
						"ship": {
							Patches: map[string]string{
								"deployment.yaml": "foo/bar/baz",
							},
							Resources: map[string]string{
								"resource.yaml": "hi",
							},
						},
					},
				},
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Patches: map[string]string{},
						Resources: map[string]string{
							"resource.yaml": "hi",
						},
					},
				},
			},
		},
		{
			Name: "delete resource, nil patches",
			DeleteFileParams: deleteFileParams{
				pathQueryParam: "resource.yaml",
				getFiles: func(overlay state.Overlay) map[string]string {
					return overlay.Resources
				},
			},
			InState: state.V1{
				Kustomize: &state.Kustomize{
					Overlays: map[string]state.Overlay{
						"ship": {
							Patches: nil,
							Resources: map[string]string{
								"resource.yaml":      "bye",
								"dont-touch-me.yaml": "forever",
							},
						},
					},
				},
			},
			ExpectState: state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Patches: nil,
						Resources: map[string]string{
							"dont-touch-me.yaml": "forever",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			mc := gomock.NewController(t)
			fakeState := mockstate.NewMockManager(mc)
			testLogger := &logger.TestLogger{T: t}
			progressmap := &daemontypes.ProgressMap{}

			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StateManager: fakeState,
				StepProgress: progressmap,
			}

			fakeState.EXPECT().CachedState().Return(state.State{
				V1: &test.InState,
			}, nil).AnyTimes()

			fakeState.EXPECT().SaveKustomize(&matchers.Is{
				Test: func(v interface{}) bool {
					c, ok := v.(*state.Kustomize)
					if !ok {
						return false
					}
					diff := deep.Equal(&test.ExpectState, c)
					if len(diff) != 0 {
						fmt.Print(fmt.Sprintf("Failed diff compare with %s", strings.Join(diff, "\n")))
						return false
					}
					return true
				},
				Describe: fmt.Sprintf("expect state equal to %s", test.ExpectState),
			}).Return(nil).AnyTimes()

			err := v2.deleteFile(test.DeleteFileParams.pathQueryParam, test.DeleteFileParams.getFiles)
			req.NoError(err)
			mc.Finish()
		})
	}
}
