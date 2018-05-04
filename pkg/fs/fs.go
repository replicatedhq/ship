package fs

import (
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func FromViper(v *viper.Viper) afero.Afero {
	return afero.Afero{Fs: afero.NewOsFs()}
}
