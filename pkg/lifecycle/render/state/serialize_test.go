package state

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestTryLoad(t *testing.T) {
	// Copy any state file out of the way
	if _, err := os.Stat(Path); !os.IsNotExist(err) {
		savedStateFile := fmt.Sprintf("./ship/state.json.%d", time.Now().UTC().UnixNano())
		os.Rename(Path, savedStateFile)
		defer func() {
			os.Remove(Path)
			os.Rename(savedStateFile, Path)
		}()
	}

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
