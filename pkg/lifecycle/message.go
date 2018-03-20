package lifecycle

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
)

var _ Executor = &messageExecutor{}

type messageExecutor struct {
	step *api.Message
}

func (e *messageExecutor) Execute(ctx context.Context, runner *Runner) error {
	debug := level.Debug(log.With(runner.Logger, "step.type", "message"))

	debug.Log("event", "step.execute", "step.level", e.step.Level)

	switch e.step.Level {
	case "error":
		runner.UI.Error(e.step.Contents)
	case "warn":
		runner.UI.Warn(e.step.Contents)
	case "debug":
		runner.UI.Output(e.step.Contents)
	default:
		runner.UI.Info(e.step.Contents)
	}
	return nil
}

func (e *messageExecutor) String() string {
	return fmt.Sprintf("Message{Contents=%s}", e.step.Contents)
}
