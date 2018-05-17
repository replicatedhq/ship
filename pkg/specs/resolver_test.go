package specs

import (
	"testing"
	"os"
	"fmt"
	"time"
	"github.com/stretchr/testify/require"
)

func TestPersistYaml(t *testing.T) {
	// Copy any yaml file out of the way
	if _, err := os.Stat(Release); !os.IsNotExist(err) {
		savedReleaseFile := fmt.Sprintf("./ship/release.yml.%d", time.Now().UTC().UnixNano())
		os.Rename(Release, savedReleaseFile)
		defer func() {
			os.Remove(Release)
			os.Rename(savedReleaseFile, Release)
		}()
	}

	req := require.New(t)

	r := Resolver{}

	err := r.persistStudioSpec([]byte("shipit"))
	req.NoError(err)
}

