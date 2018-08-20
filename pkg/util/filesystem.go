package util

import (
	"fmt"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
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

func BackupIfPresent(fs afero.Afero, basePath string, logger log.Logger, ui cli.Ui) error {
	exists, err := fs.Exists(basePath)
	if err != nil {
		return errors.Wrapf(err, "check file exists")
	}
	if !exists {
		return nil
	}

	backupDest := fmt.Sprintf("%s.bak", basePath)
	ui.Warn(fmt.Sprintf("WARNING found directory %s, backing up to %s", basePath, backupDest))

	level.Info(logger).Log("step.type", "render", "event", "unpackTarget.backup.remove", "src", basePath, "dest", backupDest)
	if err := fs.RemoveAll(backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s: remove existing %s", basePath, backupDest, backupDest)
	}
	if err := fs.Rename(basePath, backupDest); err != nil {
		return errors.Wrapf(err, "backup existing dir %s to %s", basePath, backupDest)
	}
	return nil
}
