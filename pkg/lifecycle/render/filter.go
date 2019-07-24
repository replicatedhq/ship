package render

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/replicatedhq/ship/pkg/util"
)

// removes the parent dirs of asset dests, with some conditions
func removeDests(fs *afero.Afero, dests []string) error {
	dirs := map[string]bool{}

	// calculate the set of dirs that have resources in them
	// if a file does not have an extension, assume it is a dir
	// otherwise find the parent dir of the file
	for _, dest := range dests {
		if filepath.Ext(dest) != "" {
			dest = filepath.Dir(dest)
		}
		dirs[dest] = true
	}

	for dir, _ := range dirs {
		err := util.IsLegalPath(dir)
		if err != nil {
			continue
		}

		dir = filepath.Clean(dir)

		if dir != "." && dir != "" {
			err := fs.RemoveAll(dir)
			if err != nil {
				return errors.Wrapf(err, "remove dir %s", dir)
			}
		}
	}

	return nil
}
