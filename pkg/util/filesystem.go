package util

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// FindOnlySubdir finds the only subdirectory of a directory.
// TODO make this work like the description says
func FindOnlySubdir(dir string, fs afero.Afero) (string, error) {
	subDirExists := false

	files, err := fs.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "failed to read dir")
	}

	subDir := files[0]

	if len(files) == 0 {
		return "", errors.Errorf("no files found in %s", dir)
	}

	for _, file := range files {
		if file.IsDir() {
			if !subDirExists {
				subDirExists = true
				subDir = file
			} else {
				return "", errors.Errorf("multiple subdirs found in %s", dir)
			}
		}
	}

	if subDirExists {
		return filepath.Join(dir, subDir.Name()), nil
	}

	return "", errors.New("unable to find a subdirectory")
}
