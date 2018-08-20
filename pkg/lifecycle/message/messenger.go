package message

import (
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
)

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
		templates.ShipContext{
			Logger: m.Logger,
		},
	)
	return builder
}
