package message

import (
	"context"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/spf13/viper"
)

type Messenger interface {
	Execute(ctx context.Context, release *api.Release, step *api.Message) error
	WithDaemon(d config.Daemon) Messenger
}

func FromViper(v *viper.Viper) Messenger {
	if v.GetBool("headless") {
		return &CLIMessenger{
			Logger: logger.FromViper(v),
			UI:     ui.FromViper(v),
			Viper:  v,
		}
	}

	return &DaemonMessenger{
		Logger: logger.FromViper(v),
		UI:     ui.FromViper(v),
		Viper:  v,
	}
}

func (m *DaemonMessenger) WithDaemon(d config.Daemon) Messenger {
	m.Daemon = d
	return m
}

func (m *DaemonMessenger) getBuilder() templates.Builder {
	builder := templates.NewBuilder(
		templates.NewStaticContext(),
		builderContext{
			logger: m.Logger,
			viper:  m.Viper,
			daemon: m.Daemon,
		},
	)
	return builder
}
