package render

import (
	"github.com/replicatedhq/ship/pkg/util"
)

func (r *noconfigrenderer) backupIfPresent(basePath string) error {
	return util.BackupIfPresent(r.Fs, basePath, r.Logger, r.UI)
}

func (r *renderer) backupIfPresent(basePath string) error {
	return util.BackupIfPresent(r.Fs, basePath, r.Logger, r.UI)
}
