package state

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-test/deep"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSerialize(t *testing.T) {
	templateContext := make(map[string]interface{})
	templateContext["key"] = "value"

	state := Manager{
		Logger: log.NewNopLogger(),
		FS:     afero.Afero{Fs: afero.NewMemMapFs()},
		V:      viper.New(),
	}

	req := require.New(t)

	err := state.Serialize(nil, api.ReleaseMetadata{}, templateContext)
	req.NoError(err)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		contents  string
		expect    map[string]interface{}
		expectErr error
	}{
		{
			name:     "v0 empty",
			contents: ``,
			expect:   make(map[string]interface{}),
		},
		{
			name:     "v0 empty object",
			contents: `{}`,
			expect:   make(map[string]interface{}),
		},
		{
			name:     "v0 single item",
			contents: `{"foo": "bar"}`,
			expect: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			name:     "v1 single item",
			contents: `{"v1": {"config": {"foo": "bar"}}}`,
			expect: map[string]interface{}{
				"foo": "bar",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			if test.contents != "" {
				err := fs.WriteFile(".ship/state.json", []byte(test.contents), 0777)
				req.NoError(err, "write existing state")
			}

			manager := &Manager{
				Logger: &logger.TestLogger{T: t},
				FS:     fs,
				V:      viper.New(),
			}

			state, err := manager.TryLoad()
			req.NoError(err)
			diff := deep.Equal(test.expect, state.CurrentConfig())
			req.Empty(diff)
		})
	}
}
