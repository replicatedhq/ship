package unfork

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
)

func (l *Unforker) rebuildListYaml(lists []util.List, kustomizedYamlFiles []util.PostKustomizeFile) ([]util.PostKustomizeFile, error) {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.unforker", "method", "rebuildListYaml"))

	return util.RebuildListYaml(debug, lists, kustomizedYamlFiles)
}

func (l *Unforker) writePostKustomizeFiles(step api.Unfork, postKustomizeFiles []util.PostKustomizeFile) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.unforker", "method", "writePostKustomizeFiles"))

	return util.WritePostKustomizeFiles(debug, l.FS, step.Dest, postKustomizeFiles)
}
