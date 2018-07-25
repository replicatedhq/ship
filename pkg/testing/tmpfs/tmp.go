// in some cases we have to use a real os fs and not a mem map because of how Afero handles readdir on a file
package tmpfs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func Tmpdir(t *testing.T) (string, func()) {
	req := require.New(t)
	d, err := ioutil.TempDir("/tmp", "gotest")
	req.NoError(err)

	return d, func() {
		os.RemoveAll(d)
	}

}

func Tmpfs(t *testing.T) (afero.Afero, func()) {
	dir, cleanup := Tmpdir(t)
	fs := afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), dir),
	}
	return fs, cleanup
}
