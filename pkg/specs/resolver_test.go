package specs

import (
	"testing"
	"github.com/go-kit/kit/log"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"reflect"
)

func TestPersistSpec(t *testing.T) {

	r := &Resolver{
		StateManager: &state.StateManager{
			Logger: log.NewNopLogger(),
			FS:     afero.Afero{Fs: afero.NewMemMapFs()},
		},
	}

	req := require.New(t)

	desiredSpec := []byte("my cool spec")
	err := r.persistSpec(desiredSpec)
	req.NoError(err)

	persistedSpec, err := r.StateManager.FS.ReadFile(".ship/release.yml")
	req.True(reflect.DeepEqual(desiredSpec, persistedSpec))
}
