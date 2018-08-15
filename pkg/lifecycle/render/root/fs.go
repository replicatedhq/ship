package root

import (
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
)

// Fs is a struct for a filesystem with a base path of RootPath
type Fs struct {
	afero.Afero
	RootPath string
}

// NewRootFS initializes a Fs struct with a base path of root
func NewRootFS(root string) Fs {
	if root == "" {
		root = constants.InstallerPrefixPath
	}
	return Fs{
		Afero: afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), root),
		},
		RootPath: root,
	}
}
