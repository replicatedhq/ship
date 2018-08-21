package root

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
)

// Fs is a struct for a filesystem with a base path of RootPath
type Fs struct {
	afero.Afero
	RootPath string
}

func (f Fs) TempDir(prefix, name string) (string, error) {
	if prefix == "" {
		return "", errors.New("rootfs does not support using system default temp dirs")
	}
	return f.Afero.TempDir(prefix, name)
}

// NewRootFS initializes a Fs struct with a base path of root
func NewRootFS(fs afero.Afero, root string) Fs {
	if root == "" {
		root = constants.InstallerPrefixPath
	}
	if root != "." {
		fs = afero.Afero{
			Fs: afero.NewBasePathFs(fs.Fs, root),
		}
	}
	return Fs{
		Afero:    fs,
		RootPath: root,
	}
}
