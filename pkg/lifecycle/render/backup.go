package render

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/afero"
	"github.com/pkg/errors"
)

func (r *noconfigrenderer) backupIfPresent(basePath string) error {
	return backupIfPresent(r.Fs, basePath, r.Logger)
}

func (r *renderer) backupIfPresent(basePath string) error {
	return backupIfPresent(r.Fs, basePath, r.Logger)
}

func backupIfPresent(fs afero.Afero, basePath string, logger log.Logger) error {
	exists, err := fs.Exists(basePath)
	if err != nil {
		return errors.Wrapf(err, "check file exists")
	}
	if !exists {
		return nil
	}
	backupDest := fmt.Sprintf("%s.bak", basePath)
	level.Info(logger).Log("step.type", "render", "event", "unpackTarget.backup.remove", "src", basePath, "dest", backupDest)
	if err := fs.RemoveAll(backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s: remove existing %s", basePath, backupDest, backupDest)
	}
	if err := fs.Rename(basePath, backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s", basePath, backupDest)
	}
	return nil
}

