package kubectl

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/headless"
	"github.com/stretchr/testify/require"
)

func TestForkKubectl_getBuilder(t *testing.T) {
	tests := []struct {
		name  string
		state map[string]interface{}
		meta  api.ReleaseMetadata
		init  string
		want  string
	}{
		{
			name:  "empty",
			state: map[string]interface{}{},
			meta:  api.ReleaseMetadata{},
			init:  "abc123",
			want:  "abc123",
		},
		{
			name:  "config value",
			state: map[string]interface{}{"abc": "123"},
			meta:  api.ReleaseMetadata{},
			init:  `{{repl ConfigOption "abc"}}`,
			want:  "123",
		},
		{
			name:  "metadata",
			state: map[string]interface{}{"abc": "123"},
			meta:  api.ReleaseMetadata{Semver: "abc123"},
			init:  `{{repl Installation "semver"}}`,
			want:  "abc123",
		},
		{
			name:  "static",
			state: map[string]interface{}{"abc": "123"},
			meta:  api.ReleaseMetadata{Semver: "abc123"},
			init:  `{{repl Add 1 2}}`,
			want:  "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			demon := headless.HeadlessDaemon{
				ResolvedConfig: tt.state,
			}

			k := &ForkKubectl{
				Daemon: &demon,
			}
			gotBuilder := k.getBuilder(tt.meta)

			gotString, err := gotBuilder.String(tt.init)
			req.NoError(err)
			req.Equal(tt.want, gotString)
		})
	}
}
