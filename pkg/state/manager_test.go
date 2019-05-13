package state

import (
	"fmt"
	"sync"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-test/deep"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/util"
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

			manager := NewManager(&logger.TestLogger{T: t}, fs, viper.New())

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

			manager := NewManager(&logger.TestLogger{T: t}, fs, viper.New())

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
				Logger: &logger.TestLogger{T: t},
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
				Logger: &logger.TestLogger{T: t},
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
				Logger: &logger.TestLogger{T: t},
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
					Metadata: &Metadata{
						ApplicationType: "mock application type",
						ReleaseNotes:    "",
						Version:         "test version",
						Icon:            "test icon",
						Name:            "test name",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.SerializeShipMetadata(tt.Metadata, "mock application type")
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
					Lifecycle: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: &logger.TestLogger{T: t},
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

func TestMManager_ParallelUpdates(t *testing.T) {
	tests := []struct {
		name      string
		runners   []func(*MManager, *require.Assertions, *sync.WaitGroup)
		validator func(VersionedState, *require.Assertions)
	}{
		{
			name: "lists",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// add the integers 1-20 to the list
					for i := 1; i <= 20; i++ {
						err := m.SerializeListsMetadata(util.List{APIVersion: fmt.Sprintf("%d", i)})
						req.NoError(err)
					}
					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				req.Len(state.V1.Metadata.Lists, 20)
			},
		},
		{
			name: "emptied lists",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					err := m.ClearListsMetadata()
					req.NoError(err)

					// add the integers 1-20 to the list
					for i := 1; i <= 20; i++ {
						err := m.SerializeListsMetadata(util.List{APIVersion: fmt.Sprintf("%d", i)})
						req.NoError(err)
					}

					err = m.ClearListsMetadata()
					req.NoError(err)

					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				req.Len(state.V1.Metadata.Lists, 0)
			},
		},
		{
			name: "lists and app metadata",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// add the integers 1-20 to the list
					for i := 1; i <= 20; i++ {
						err := m.SerializeListsMetadata(util.List{APIVersion: fmt.Sprintf("%d", i)})
						req.NoError(err)
					}
					group.Done()
				},
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					err := m.SerializeAppMetadata(api.ReleaseMetadata{Semver: "tested"})
					req.NoError(err)
					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				req.Len(state.V1.Metadata.Lists, 20)
				req.Equal("tested", state.V1.Metadata.Version)
			},
		},
		{
			name: "lists, release name and namespace",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// add the integers 1-20 to the list
					for i := 1; i <= 20; i++ {
						err := m.SerializeListsMetadata(util.List{APIVersion: fmt.Sprintf("%d", i)})
						req.NoError(err)
					}
					group.Done()
				},
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					err := m.SerializeReleaseName("testedName")
					req.NoError(err)
					group.Done()
				},
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					err := m.SerializeNamespace("testedNS")
					req.NoError(err)
					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				req.Len(state.V1.Metadata.Lists, 20)
				req.Equal("testedName", state.CurrentReleaseName())
				req.Equal("testedNS", state.CurrentNamespace())
			},
		},
		{
			name: "lists and upstream",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				// lists
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// add the integers 1-20 to the list
					for i := 1; i <= 20; i++ {
						err := m.SerializeListsMetadata(util.List{APIVersion: fmt.Sprintf("%d", i)})
						req.NoError(err)
					}
					group.Done()
				},
				// first upstream mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append the integers 1-200 to the upstream
					for i := 1; i <= 200; i++ {
						_, err := m.StateUpdate(func(state VersionedState) (VersionedState, error) {
							state.V1.Upstream += fmt.Sprintf(" a:%d ", i)
							return state, nil
						})
						req.NoError(err)
					}
					group.Done()
				},
				// second upstream mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append the integers 1-200 to the upstream
					for i := 1; i <= 200; i++ {
						_, err := m.StateUpdate(func(state VersionedState) (VersionedState, error) {
							state.V1.Upstream += fmt.Sprintf(" b:%d ", i)
							return state, nil
						})
						req.NoError(err)
					}
					group.Done()
				},
				// third upstream mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append the integers 1-200 to the upstream
					for i := 1; i <= 200; i++ {
						_, err := m.StateUpdate(func(state VersionedState) (VersionedState, error) {
							state.V1.Upstream += fmt.Sprintf(" c:%d ", i)
							return state, nil
						})
						req.NoError(err)
					}
					group.Done()
				},
				// fourth upstream mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append the integers 1-200 to the upstream
					for i := 1; i <= 200; i++ {
						_, err := m.StateUpdate(func(state VersionedState) (VersionedState, error) {
							state.V1.Upstream += fmt.Sprintf(" d:%d ", i)
							return state, nil
						})
						req.NoError(err)
					}
					group.Done()
				},
				// fifth upstream mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append the integers 1-200 to the upstream
					for i := 1; i <= 200; i++ {
						_, err := m.StateUpdate(func(state VersionedState) (VersionedState, error) {
							state.V1.Upstream += fmt.Sprintf(" e:%d ", i)
							return state, nil
						})
						req.NoError(err)
					}
					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				req.Len(state.V1.Metadata.Lists, 20)

				totalUpstream := state.Upstream()
				for _, str := range []string{"a", "b", "c", "d", "e"} {
					for i := 1; i <= 200; i++ {
						req.Contains(totalUpstream, fmt.Sprintf(" %s:%d ", str, i))
					}
				}
			},
		},
		{
			name: "certs and keys",
			runners: []func(*MManager, *require.Assertions, *sync.WaitGroup){
				// first cert mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append 100 certs to the cert list
					for i := 1; i <= 100; i++ {
						err := m.AddCert(fmt.Sprintf(" a:%d ", i), util.CertType{})
						req.NoError(err)
					}
					group.Done()
				},
				// second cert mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append 100 certs to the cert list
					for i := 1; i <= 100; i++ {
						err := m.AddCert(fmt.Sprintf(" b:%d ", i), util.CertType{})
						req.NoError(err)
					}
					group.Done()
				},
				// third cert mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append 100 certs to the cert list
					for i := 1; i <= 100; i++ {
						err := m.AddCert(fmt.Sprintf(" c:%d ", i), util.CertType{})
						req.NoError(err)
					}
					group.Done()
				},
				// first ca mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append 100 CAs to the CA list
					for i := 1; i <= 100; i++ {
						err := m.AddCA(fmt.Sprintf(" a:%d ", i), util.CAType{})
						req.NoError(err)
					}
					group.Done()
				},
				// second ca mutator
				func(m *MManager, req *require.Assertions, group *sync.WaitGroup) {
					// append 100 CAs to the CA list
					for i := 1; i <= 100; i++ {
						err := m.AddCA(fmt.Sprintf(" b:%d ", i), util.CAType{})
						req.NoError(err)
					}
					group.Done()
				},
			},
			validator: func(state VersionedState, req *require.Assertions) {
				totalCAs := state.CurrentCAs()
				for _, str := range []string{"a", "b"} {
					for i := 1; i <= 100; i++ {
						req.Contains(totalCAs, fmt.Sprintf(" %s:%d ", str, i))
					}
				}
				totalCerts := state.CurrentCerts()
				for _, str := range []string{"a", "b", "c"} {
					for i := 1; i <= 100; i++ {
						req.Contains(totalCerts, fmt.Sprintf(" %s:%d ", str, i))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := require.New(t)
			m := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			initialState := VersionedState{V1: &V1{Lifecycle: nil}}

			group := sync.WaitGroup{}

			err := m.serializeAndWriteState(initialState)
			req.NoError(err)

			group.Add(len(tt.runners))
			for _, runner := range tt.runners {
				go runner(m, req, &group)
			}

			group.Wait()
			actualState, err := m.TryLoad()
			req.NoError(err)

			tt.validator(actualState.Versioned(), req)
		})
	}
}

func TestMManager_AddCA(t *testing.T) {
	tests := []struct {
		name     string
		caName   string
		newCA    util.CAType
		wantErr  bool
		before   VersionedState
		expected VersionedState
	}{
		{
			name:   "basic test",
			caName: "aCA",
			newCA:  util.CAType{Cert: "aCert", Key: "aKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					CAs: map[string]util.CAType{
						"aCA": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
		},
		{
			name:   "add to existing",
			caName: "bCA",
			newCA:  util.CAType{Cert: "bCert", Key: "bKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					CAs: map[string]util.CAType{
						"aCA": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					CAs: map[string]util.CAType{
						"aCA": {Cert: "aCert", Key: "aKey"},
						"bCA": {Cert: "bCert", Key: "bKey"},
					},
				},
			},
		},
		{
			name:    "colliding ca names",
			wantErr: true,
			caName:  "aCA",
			newCA:   util.CAType{Cert: "aCert", Key: "aKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					CAs: map[string]util.CAType{
						"aCA": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					CAs: map[string]util.CAType{
						"aCA": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.AddCA(tt.caName, tt.newCA)
			if !tt.wantErr {
				req.NoError(err, "MManager.AddCA() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}

func TestMManager_AddCert(t *testing.T) {
	tests := []struct {
		name     string
		certName string
		newCert  util.CertType
		wantErr  bool
		before   VersionedState
		expected VersionedState
	}{
		{
			name:     "basic test",
			certName: "aCert",
			newCert:  util.CertType{Cert: "aCert", Key: "aKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					Certs: map[string]util.CertType{
						"aCert": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
		},
		{
			name:     "add to existing",
			certName: "bCert",
			newCert:  util.CertType{Cert: "bCert", Key: "bKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					Certs: map[string]util.CertType{
						"aCert": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					Certs: map[string]util.CertType{
						"aCert": {Cert: "aCert", Key: "aKey"},
						"bCert": {Cert: "bCert", Key: "bKey"},
					},
				},
			},
		},
		{
			name:     "colliding ca names",
			wantErr:  true,
			certName: "aCert",
			newCert:  util.CertType{Cert: "aCert", Key: "aKey"},
			before: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					Certs: map[string]util.CertType{
						"aCert": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
			expected: VersionedState{
				V1: &V1{
					Upstream: "abc123",
					Certs: map[string]util.CertType{
						"aCert": {Cert: "aCert", Key: "aKey"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			m := &MManager{
				Logger: &logger.TestLogger{T: t},
				FS:     afero.Afero{Fs: afero.NewMemMapFs()},
				V:      viper.New(),
			}

			err := m.serializeAndWriteState(tt.before)
			req.NoError(err)

			err = m.AddCert(tt.certName, tt.newCert)
			if !tt.wantErr {
				req.NoError(err, "MManager.AddCert() error = %v", err)
			} else {
				req.Error(err)
			}

			actualState, err := m.TryLoad()
			req.NoError(err)

			req.Equal(tt.expected, actualState)
		})
	}
}
