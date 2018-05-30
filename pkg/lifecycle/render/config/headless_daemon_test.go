package config

import (
	"testing"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type TestHeadless struct {
	Name     string
	Config   []byte
	Expected map[string]interface{}
}

func TestHeadlessDaemon(t *testing.T) {
	tests := []TestHeadless{
		{
			Name:     "basic",
			Config:   []byte(`{"spam": "eggs"}`),
			Expected: map[string]interface{}{"spam": "eggs"},
		},
		{
			Name:     "basic",
			Config:   []byte(`{"spam": "eggs"}`),
			Expected: map[string]interface{}{"spam": "eggs"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
			err := fakeFS.WriteFile(".ship/state.json", test.Config, 0666)
			req.NoError(err)

			testLogger := &logger.TestLogger{T: t}
			daemon := &HeadlessDaemon{
				StateManager: &state.StateManager{
					Logger: testLogger,
					FS:     fakeFS,
				},
				Logger: testLogger,
			}

			cfg := daemon.GetCurrentConfig()
			req.Equal(cfg, test.Expected)
		})
	}
}
