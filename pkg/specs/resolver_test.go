package specs

import (
	"reflect"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestPersistSpec(t *testing.T) {

	r := &Resolver{
		StateManager: &state.Manager{
			Logger: log.NewNopLogger(),
			FS:     afero.Afero{Fs: afero.NewMemMapFs()},
			V:      viper.New(),
		},
	}

	req := require.New(t)

	desiredSpec := []byte("my cool spec")
	err := r.persistSpec(desiredSpec)
	req.NoError(err)

	persistedSpec, err := r.StateManager.FS.ReadFile(".ship/release.yml")
	req.True(reflect.DeepEqual(desiredSpec, persistedSpec))
}
