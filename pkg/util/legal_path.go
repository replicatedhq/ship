package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// IsLegalPath checks if the provided path is a relative path within the current working directory. If it is not, it returns an error.
func IsLegalPath(path string) error {

	if filepath.IsAbs(path) {
		return fmt.Errorf("cannot write to an absolute path: %s", path)
	}

	relPath, err := filepath.Rel(".", path)
	if err != nil {
		return errors.Wrap(err, "find relative path to dest")
	}

	if strings.Contains(relPath, "..") {
		return fmt.Errorf("cannot write to a path that is a parent of the working dir: %s", relPath)
	}

	return nil
}
