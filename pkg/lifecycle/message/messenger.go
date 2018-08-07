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
	daemonless DaemonlessMessenger,
) lifecycle.Messenger {
	if v.GetBool("headless") {
		return &cli
	} else if v.GetBool("navigate-lifecycle") { // opt in feature flag for v2 routing/lifecycle rules
		return &daemonless
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
