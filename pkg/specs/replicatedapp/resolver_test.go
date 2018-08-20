package replicatedapp

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPersistSpec(t *testing.T) {

	r := &resolver{
		FS: afero.Afero{Fs: afero.NewMemMapFs()},
	}

	req := require.New(t)

	desiredSpec := []byte("my cool spec")
	err := r.persistSpec(desiredSpec)
	req.NoError(err)

	persistedSpec, err := r.FS.ReadFile(constants.ReleasePath)
	req.True(reflect.DeepEqual(desiredSpec, persistedSpec))
}
