package kustomize

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
)

func (l *Kustomizer) writePostKustomizeFiles(step api.Kustomize, postKustomizeFiles []util.PostKustomizeFile) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "writePostKustomizeFiles"))

	return util.WritePostKustomizeFiles(debug, l.FS, step.Dest, postKustomizeFiles)
}
