package specs

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPersistSpec(t *testing.T) {

	r := &Resolver{
		StateManager: &state.StateManager{
			Logger: log.NewNopLogger(),
			FS:     afero.Afero{Fs: afero.NewMemMapFs()},
		},
	}

	// Copy any release file out of the way
	if _, err := r.StateManager.FS.ReadFile(ReleasePath); err != nil {
		savedSpec := fmt.Sprintf("./ship/release.yml.%d", time.Now().UTC().UnixNano())
		r.StateManager.FS.Rename(ReleasePath, savedSpec)
		defer func() {
			r.StateManager.FS.Remove(ReleasePath)
			r.StateManager.FS.Rename(savedSpec, ReleasePath)
		}()
	}

	req := require.New(t)

	err := r.persistSpec([]byte("my cool spec"))
	req.NoError(err)
}
