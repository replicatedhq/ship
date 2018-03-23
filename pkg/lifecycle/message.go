package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
)

type messenger struct {
	Logger log.Logger
	UI     cli.Ui
}

func (e *messenger) Execute(ctx context.Context, step *api.Message) error {
	debug := level.Debug(log.With(e.Logger, "step.type", "message"))

	debug.Log("event", "step.execute", "step.level", step.Level)

	switch step.Level {
	case "error":
		e.UI.Error(step.Contents)
	case "warn":
		e.UI.Warn(step.Contents)
	case "debug":
		e.UI.Output(step.Contents)
	default:
		e.UI.Info(step.Contents)
	}
	return nil
}
