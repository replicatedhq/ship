package statusonly

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
)

// StatusReceiver implements the daemontypes statusreceiver by sending progress
// messages on a channel
var _ daemontypes.StatusReceiver = &StatusReceiver{}

type StatusReceiver struct {
	Logger     log.Logger
	Name       string
	OnProgress func(daemontypes.Progress)
}

func (d *StatusReceiver) SetStepName(ctx context.Context, name string) {
	d.OnProgress(daemontypes.StringProgress("phase", name))
}

func (d *StatusReceiver) SetProgress(progress daemontypes.Progress) {
	d.OnProgress(progress)
}

func (d *StatusReceiver) ClearProgress() {
	// no-op I think, we'll implement this if/when we need it
}

func (d *StatusReceiver) PushStreamStep(ctx context.Context, messages <-chan daemontypes.Message) {
	debug := level.Debug(log.With(d.Logger, "method", "pushStreamStep"))
	select {
	case <-ctx.Done():
		debug.Log("event", "ctx.Done", "err", ctx.Err())
	case msg := <-messages:
		debug.Log("event", "message.receive", "contents", fmt.Sprintf("%.32s", msg.Contents))
		d.OnProgress(daemontypes.MessageProgress(d.Name, msg))
	}
}

func (d *StatusReceiver) PushMessageStep(ctx context.Context, step daemontypes.Message, actions []daemontypes.Action) {
	debug := level.Debug(log.With(d.Logger, "method", "pushMessageStep"))

	debug.Log("event", "message")
	d.OnProgress(daemontypes.JSONProgress("message step", map[string]interface{}{
		"status":  "message",
		"message": step,
		"actions": actions,
	}))
}
