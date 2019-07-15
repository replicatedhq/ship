package kustomize

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
)

func TestKustomizer_maybeCleanupKustomizeState(t *testing.T) {

	tests := []struct {
		name          string
		InKust        *state.Kustomize
		keepStateFlag bool
		WantKust      *state.Kustomize
		wantErr       bool
	}{
		{
			name:     "nil input state",
			InKust:   nil,
			WantKust: nil,
		},
		{
			name: "no excluded files",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:   map[string]string{"abc": "xyz"},
						Resources: map[string]string{"abc": "xyz"},
					},
				},
			},
			WantKust: nil,
		},
		{
			name: "excluded files",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
		},
		{
			name: "keep state",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
			keepStateFlag: true,
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			tViper := viper.New()
			tViper.Set(constants.FilesInStateFlag, tt.keepStateFlag)

			manager, err := state.NewDisposableManager(log.NewNopLogger(), fs, tViper)
			req.NoError(err)

			err = manager.SaveKustomize(tt.InKust)
			req.NoError(err)

			l := &Kustomizer{
				State:  manager,
				Viper:  tViper,
				Logger: log.NewNopLogger(),
			}

			err = l.maybeCleanupKustomizeState()
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			outState, err := manager.CachedState()
			req.NoError(err)

			outKust := outState.CurrentKustomize()
			req.Equal(tt.WantKust, outKust)
		})
	}
}
