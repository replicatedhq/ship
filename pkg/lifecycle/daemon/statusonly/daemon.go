package statusonly

import (
	"context"

	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

// StatusReceiver implements the daemontypes statusreceiver by sending progress
// messages on a channel
var _ daemontypes.StatusReceiver = &StatusReceiver{}

type StatusReceiver struct {
}

func (d *StatusReceiver) SetStepName(context.Context, string) {
	panic("implement me")
}

func (d *StatusReceiver) SetProgress(daemontypes.Progress) {
	panic("implement me")
}

func (d *StatusReceiver) ClearProgress() {
	panic("implement me")
}

func (d *StatusReceiver) PushStreamStep(context.Context, <-chan daemontypes.Message) {
	panic("implement me")
}
