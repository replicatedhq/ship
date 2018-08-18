package util

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// FindOnlySubdir finds the only subdirectory of a directory.
// TODO make this work like the description says
func FindOnlySubdir(dir string, filesystem afero.Fs) (string, error) {
	fs := afero.Afero{Fs: filesystem}
	files, err := fs.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "failed to read dir")
	}

	firstFoundFile := files[0]
	if !firstFoundFile.IsDir() {
		return "", errors.New(fmt.Sprintf("unable to find subdirectory, found file %s instead", firstFoundFile.Name()))
	}
	return filepath.Join(dir, firstFoundFile.Name()), nil
}
