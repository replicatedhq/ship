package helmValues

import (
	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

func NewDaemonlessHelmValues(
	fs afero.Afero,
	logger log.Logger,
	stateManager state.Manager,
) lifecycle.HelmValues {
	return &daemonlessHelmValues{
		Fs:           fs,
		Logger:       logger,
		StateManager: stateManager,
	}
}

func (h *daemonlessHelmValues) resolveStateHelmValues() error {
	return resolveStateHelmValues(h.Logger, h.StateManager, h.Fs)
}
