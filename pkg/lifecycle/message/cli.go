package message

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

var _ lifecycle.Messenger = &CLIMessenger{}

type CLIMessenger struct {
	dig.In

	Logger         log.Logger
	UI             cli.Ui
	Viper          *viper.Viper
	BuilderBuilder *templates.BuilderBuilder
}

func (m *CLIMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	debug := level.Debug(log.With(m.Logger, "step.type", "message"))

	debug.Log("event", "step.execute", "step.level", step.Level)

	builder, err := m.BuilderBuilder.BaseBuilder(release.Metadata)
	if err != nil {
		return errors.Wrap(err, "get builder")
	}

	built, _ := builder.String(step.Contents)

	switch step.Level {
	case "error":
		m.UI.Error(fmt.Sprintf("\n%s", built))
	case "warn":
		m.UI.Warn(fmt.Sprintf("\n%s", built))
	case "debug":
		m.UI.Output(fmt.Sprintf("\n%s", built))
	default:
		m.UI.Info(fmt.Sprintf("\n%s", built))
	}
	return nil
}
