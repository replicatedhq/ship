package config

import (
	"testing"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestHeadlessDaemon(t *testing.T) {
	req := require.New(t)

	fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
	err := fakeFS.WriteFile(".ship/state.json", []byte(`{"spam": "eggs"}`), 0666)
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
	req.Equal("eggs", cfg["spam"])
}
