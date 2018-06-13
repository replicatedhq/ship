package message

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

type Messenger interface {
	Execute(ctx context.Context, release *api.Release, step *api.Message) error
	WithDaemon(d config.Daemon) Messenger
}

func NewMessenger(
	v *viper.Viper,
	cli CLIMessenger,
	daemon DaemonMessenger,
) Messenger {
	if v.GetBool("headless") {
		return &cli
	}

	return &daemon
}

func (m *DaemonMessenger) WithDaemon(d config.Daemon) Messenger {
	m.Daemon = d
	return m
}

func (m *DaemonMessenger) getBuilder(meta api.ReleaseMetadata) templates.Builder {
	builder := m.BuilderBuilder.NewBuilder(
		m.BuilderBuilder.NewStaticContext(),
		builderContext{
			logger: m.Logger,
			viper:  m.Viper,
			daemon: m.Daemon,
		},
		&templates.InstallationContext{
			Meta:  meta,
			Viper: m.Viper,
		},
	)
	return builder
}
