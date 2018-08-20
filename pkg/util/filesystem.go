package util

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// FindOnlySubdir finds the only subdirectory of a directory.
// TODO make this work like the description says
func FindOnlySubdir(dir string, fs afero.Afero) (string, error) {
	subDirExists := false
	firstSubDirIndex := 0

	files, err := fs.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "failed to read dir")
	}

	if len(files) == 0 {
		return "", errors.New(fmt.Sprintf("no files found in %s", dir))
	}

	for idx, file := range files {
		if file.IsDir() {
			if !subDirExists {
				subDirExists = true
				firstSubDirIndex = idx
			} else {
				return "", errors.New(fmt.Sprintf("multiple subdirs found in %s", dir))
			}
		}
	}

	if subDirExists {
		return filepath.Join(dir, files[firstSubDirIndex].Name()), nil
	}

	return "", errors.New("unable to find a subdirectory")
}
