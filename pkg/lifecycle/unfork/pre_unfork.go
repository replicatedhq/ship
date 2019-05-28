package unfork

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
)

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func (l *Unforker) PreExecute(ctx context.Context, step api.Step) error {
	// Split multi doc forked base first as it will be unmarshalled incorrectly in the following steps
	if err := l.maybeSplitMultidocYaml(ctx, step.Unfork.ForkedBase); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml forked base")
	}

	if err := l.maybeSplitMultidocYaml(ctx, step.Unfork.UpstreamBase); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml upstream base")
	}

	return nil
}
