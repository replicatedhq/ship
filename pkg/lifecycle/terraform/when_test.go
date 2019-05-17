package terraform

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestEvaluateWhen(t *testing.T) {
	tests := []struct {
		name    string
		State   state.State
		when    string
		release api.Release
		want    bool
	}{
		{
			name:    "no when",
			State:   state.State{V1: &state.V1{}},
			when:    "",
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    true,
		},
		{
			name:    "true when",
			State:   state.State{V1: &state.V1{}},
			when:    "true",
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    true,
		},
		{
			name:    "false when",
			State:   state.State{V1: &state.V1{}},
			when:    "false",
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    false,
		},
		{
			name:    "trivial template when true",
			State:   state.State{V1: &state.V1{}},
			when:    "{{repl eq 1 1}}",
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    true,
		},
		{
			name:    "trivial template when false",
			State:   state.State{V1: &state.V1{}},
			when:    "{{repl eq 1 2}}",
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    false,
		},
		{
			name:    "configOption template when true",
			State:   state.State{V1: &state.V1{Config: map[string]interface{}{"theOption": "hello_world"}}},
			when:    `{{repl ConfigOptionEquals "theOption" "hello_world"}}`,
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    true,
		},
		{
			name:    "configOption template when false",
			State:   state.State{V1: &state.V1{Config: map[string]interface{}{"theOption": "hello_world"}}},
			when:    `{{repl ConfigOptionEquals "theOption" "something else"}}`,
			release: api.Release{Metadata: api.ReleaseMetadata{}},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			tlogger := &logger.TestLogger{T: t}
			tviper := viper.New()
			stateManager := state.NewManager(tlogger, fs, tviper)

			req.NoError(stateManager.Save(tt.State))
			req.NoError(stateManager.SerializeAppMetadata(tt.release.Metadata))

			forkTF := &ForkTerraformer{
				Logger:       tlogger,
				Viper:        tviper,
				StateManager: stateManager,
			}
			daemonlessTF := &DaemonlessTerraformer{
				Logger:       tlogger,
				Viper:        tviper,
				StateManager: stateManager,
			}

			req.Equal(tt.want, forkTF.evaluateWhen(tt.when, tt.release), "ForkTerraformer.evaluateWhen()")
			req.Equal(tt.want, daemonlessTF.evaluateWhen(tt.when, tt.release), "DaemonlessTerraformer.evaluateWhen()")
		})
	}
}
