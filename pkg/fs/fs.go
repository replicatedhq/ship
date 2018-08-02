package fs

import (
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// NewBaseFilesystem creates a new Afero OS filesystem
func NewBaseFilesystem() afero.Afero {
	return afero.Afero{Fs: afero.NewOsFs()}
}

// FilesystemParams is a struct that contains Filesystem configuration
type FilesystemParams struct {
	AssetsPath  string
	DotShipPath string
}

// NewFilesystemParams creates a new FilesystemParams config object
func NewFilesystemParams(v *viper.Viper) FilesystemParams {
	assetsPath := ""
	if v.GetBool("is-app") {
		assetsPath = constants.InstallerPrefixPath
	}

	return FilesystemParams{
		AssetsPath:  assetsPath,
		DotShipPath: constants.ShipPath,
	}
}

// Filesystems is a struct that returns multiple filesystems for use
// in ship execution
type Filesystems struct {
	DotShip afero.Afero
	Assets  afero.Afero
}

// NewFilesystems creates a new Filesystems struct for use in ship execution
func NewFilesystems(fsp FilesystemParams, baseFilesystem afero.Afero) Filesystems {
	return Filesystems{
		DotShip: afero.Afero{
			Fs: afero.NewBasePathFs(baseFilesystem.Fs, fsp.DotShipPath),
		},
		Assets: afero.Afero{
			Fs: afero.NewBasePathFs(baseFilesystem.Fs, fsp.AssetsPath),
		},
	}
}
