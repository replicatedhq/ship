package message

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/spf13/viper"
)

var _ Messenger = &CLIMessenger{}

type CLIMessenger struct {
	Logger log.Logger
	UI     cli.Ui
	Viper  *viper.Viper
}

func (m *CLIMessenger) WithDaemon(_ config.Daemon) Messenger {
	return m
}

func (e *CLIMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	debug := level.Debug(log.With(e.Logger, "step.type", "message"))

	debug.Log("event", "step.execute", "step.level", step.Level)

	builder := e.getBuilder(release)
	built, _ := builder.String(step.Contents)

	switch step.Level {
	case "error":
		e.UI.Error(fmt.Sprintf("\n%s", built))
	case "warn":
		e.UI.Warn(fmt.Sprintf("\n%s", built))
	case "debug":
		e.UI.Output(fmt.Sprintf("\n%s", built))
	default:
		e.UI.Info(fmt.Sprintf("\n%s", built))
	}
	return nil
}

func (e *CLIMessenger) getBuilder(release *api.Release) templates.Builder {
	builder := templates.NewBuilder(
		templates.NewStaticContext(),
		builderContext{
			logger: e.Logger,
			viper:  e.Viper,
		},
		&templates.InstallationContext{
			Meta:  release.Metadata,
			Viper: e.Viper,
		},
	)
	return builder
}
