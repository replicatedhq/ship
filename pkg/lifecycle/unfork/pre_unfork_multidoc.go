package unfork

import (
	"context"

	"github.com/replicatedhq/ship/pkg/util"
)

// this function is not perfect, and has known limitations. One of these is that it does not account for `\n---\n` in multiline strings.
func (l *Unforker) maybeSplitMultidocYaml(ctx context.Context, localPath string) error {
	return util.MaybeSplitMultidocYaml(ctx, l.FS, localPath)
}
