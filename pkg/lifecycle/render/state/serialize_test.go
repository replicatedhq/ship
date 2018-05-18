package state

import (
	"testing"
	"os"
	"fmt"
	"time"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/replicatedcom/ship/pkg/api"
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

	state := StateManager{
		Logger: log.NewNopLogger(),
	}

	req := require.New(t)

	err := state.Serialize(nil, api.ReleaseMetadata{}, templateContext)
	req.NoError(err)
}
