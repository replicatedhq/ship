package state

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-test/deep"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSerialize(t *testing.T) {
	templateContext := make(map[string]interface{})
	templateContext["key"] = "value"

	state := &MManager{
		Logger: log.NewNopLogger(),
		FS:     afero.Afero{Fs: afero.NewMemMapFs()},
		V:      viper.New(),
	}

	req := require.New(t)

	err := state.SerializeConfig(nil, api.ReleaseMetadata{}, templateContext)
	req.NoError(err)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name            string
		contents        string
		expectConfig    map[string]interface{}
		expectKustomize *Kustomize
		expectErr       error
	}{
		{
			name:         "v0 Empty",
			contents:     ``,
			expectConfig: make(map[string]interface{}),
		},
		{
			name:         "v0 Empty object",
			contents:     `{}`,
			expectConfig: make(map[string]interface{}),
		},
		{
			name:     "v0 single item",
			contents: `{"foo": "bar"}`,
			expectConfig: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			name:     "v1 single item",
			contents: `{"v1": {"config": {"foo": "bar"}}}`,
			expectConfig: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			name: "kustomize",
			contents: `{"v1": {"kustomize": {"overlays": {
"ship": {
  "patches": {
	"deployment.yml": "some-fake-overlay"
  }
}
}}}}`,
			expectKustomize: &Kustomize{
				Overlays: map[string]Overlay{
					"ship": {
						Patches: map[string]string{
							"deployment.yml": `some-fake-overlay`,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			if test.contents != "" {
				err := fs.WriteFile(constants.StatePath, []byte(test.contents), 0777)
				req.NoError(err, "write existing state")
			}

			manager := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     fs,
				V:      viper.New(),
			}

			state, err := manager.TryLoad()
			req.NoError(err)
			if test.expectConfig != nil {
				diff := deep.Equal(test.expectConfig, state.CurrentConfig())
				req.Empty(diff)
			}

			if test.expectKustomize != nil {
				diff := deep.Equal(test.expectKustomize, state.CurrentKustomize())
				req.Empty(diff)
			}
		})
	}
}

func TestHelmValue(t *testing.T) {
	tests := []struct {
		name                  string
		chartValuesOnInit     string
		userInputValues       string
		chartValuesOnUpdate   string
		wantValuesAfterUpdate string
	}{
		{
			name:                  "override single value persists through update",
			chartValuesOnInit:     `replicas: 1`,
			userInputValues:       `replicas: 5`,
			chartValuesOnUpdate:   `replicas: 2`,
			wantValuesAfterUpdate: `replicas: 5`,
		},
		// todo fixme I fail
		//		{
		//			name: "override one value, different default changes",
		//			chartValuesOnInit: `
		//replicas: 1
		//image: nginx:stable
		//`,
		//			userInputValues: `
		//replicas: 5
		//image: nginx:stable
		//`,
		//			chartValuesOnUpdate: `
		//replicas: 2
		//image: nginx:latest
		//`,
		//			wantValuesAfterUpdate: `
		//replicas: 5
		//image: nginx:latest
		//`,
		//		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			manager := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     fs,
				V:      viper.New(),
			}

			err := manager.SerializeHelmValues(test.userInputValues, test.chartValuesOnInit)
			req.NoError(err)

			t0State, err := manager.TryLoad()
			req.NoError(err)
			req.Equal(test.userInputValues, t0State.CurrentHelmValues())

			err = manager.SerializeHelmValues(test.userInputValues, test.chartValuesOnUpdate)
			req.NoError(err)

			t1State, err := manager.TryLoad()
			req.NoError(err)
			req.Equal(test.wantValuesAfterUpdate, t1State.CurrentHelmValues())
		})
	}
}

func TestMManager_SerializeChartURL(t *testing.T) {
	tests := []struct {
		name     string
		URL      string
		wantErr  bool
		before   VersionedState
		expected VersionedState
	}{
		{
			name: "basic test",
			URL:  "abc123",
			before: VersionedState{
				V1: &V1{},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
				},
			},
		},
		{
			name: "no wipe",
			URL:  "abc123",
			before: VersionedState{
				V1: &V1{
					ChartRepoURL: "abc123_",
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream:     "abc123",
					ChartRepoURL: "abc123_",
				},
			},
		},
		{
			name: "no wipe, but still override",
			URL:  "xyz789",
			before: VersionedState{
				V1: &V1{
					ChartURL: "abc123",
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "xyz789",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: log.NewNopLogger(),
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.SerializeUpstream(tt.URL)
			if !tt.wantErr {
				req.NoError(err, "MManager.SerializeChartURL() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}

func TestMManager_SerializeContentSHA(t *testing.T) {
	tests := []struct {
		name       string
		ContentSHA string
		wantErr    bool
		before     VersionedState
		expected   VersionedState
	}{
		{
			name:       "basic test",
			ContentSHA: "abc123",
			before: VersionedState{
				V1: &V1{},
			},
			expected: VersionedState{
				V1: &V1{
					ContentSHA: "abc123",
				},
			},
		},
		{
			name:       "no wipe",
			ContentSHA: "abc123",
			before: VersionedState{
				V1: &V1{
					ChartRepoURL: "abc123_",
				},
			},
			expected: VersionedState{
				V1: &V1{
					ContentSHA:   "abc123",
					ChartRepoURL: "abc123_",
				},
			},
		},
		{
			name:       "no wipe, but still override",
			ContentSHA: "xyz789",
			before: VersionedState{
				V1: &V1{
					ContentSHA: "abc123",
				},
			},
			expected: VersionedState{
				V1: &V1{
					ContentSHA: "xyz789",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: log.NewNopLogger(),
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.SerializeContentSHA(tt.ContentSHA)
			if !tt.wantErr {
				req.NoError(err, "MManager.SerializeContentSHA() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}

func TestMManager_SerializeHelmValues(t *testing.T) {
	tests := []struct {
		name         string
		HelmValues   string
		HelmDefaults string // is discarded by the function
		wantErr      bool
		before       VersionedState
		expected     VersionedState
	}{
		{
			name:       "basic test",
			HelmValues: "abc123",
			before: VersionedState{
				V1: &V1{},
			},
			expected: VersionedState{
				V1: &V1{
					HelmValues: "abc123",
				},
			},
		},
		{
			name:       "no wipe",
			HelmValues: "abc123",
			before: VersionedState{
				V1: &V1{
					ChartRepoURL: "abc123_",
				},
			},
			expected: VersionedState{
				V1: &V1{
					HelmValues:   "abc123",
					ChartRepoURL: "abc123_",
				},
			},
		},
		{
			name:       "no wipe, but still override",
			HelmValues: "xyz789",
			before: VersionedState{
				V1: &V1{
					HelmValues: "abc123",
				},
			},
			expected: VersionedState{
				V1: &V1{
					HelmValues: "xyz789",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: log.NewNopLogger(),
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.SerializeHelmValues(tt.HelmValues, tt.HelmDefaults)
			if !tt.wantErr {
				req.NoError(err, "MManager.SerializeHelmValues() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}

func TestMManager_SerializeShipMetadata(t *testing.T) {
	tests := []struct {
		name     string
		Metadata api.ShipAppMetadata
		wantErr  bool
		before   VersionedState
		expected VersionedState
	}{
		{
			name: "basic test",
			Metadata: api.ShipAppMetadata{
				Version: "test version",
				Icon:    "test icon",
				Name:    "test name",
			},
			before: VersionedState{
				V1: &V1{},
			},
			expected: VersionedState{
				V1: &V1{
					Metadata: map[string]string{
						"version": "test version",
						"icon":    "test icon",
						"name":    "test name",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: log.NewNopLogger(),
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.SerializeShipMetadata(tt.Metadata)
			if !tt.wantErr {
				req.NoError(err, "MManager.SerializeShipMetadata() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}

func TestMManager_ResetLifecycle(t *testing.T) {
	tests := []struct {
		name     string
		before   VersionedState
		expected VersionedState
	}{
		{
			name: "basic test",
			before: VersionedState{
				V1: &V1{
					Lifecycle: &Lifeycle{
						StepsCompleted: map[string]interface{}{
							"step1": true,
							"step2": true,
							"step3": true,
						},
					},
				},
			},
			expected: VersionedState{
				V1: &V1{
					Lifecycle: &Lifeycle{
						StepsCompleted: nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: log.NewNopLogger(),
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.ResetLifecycle()
			req.NoError(err)

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}
