package message

import (
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

func NewMessenger(
	v *viper.Viper,
	cli CLIMessenger,
	daemon DaemonMessenger,
) lifecycle.Messenger {
	if v.GetBool("headless") {
		return &cli
	}

	return &daemon
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
