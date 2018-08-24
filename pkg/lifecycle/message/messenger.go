package message

import (
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
)

func (m *DaemonMessenger) getBuilder(meta api.ReleaseMetadata) (templates.Builder, error) {
	builder, err := m.BuilderBuilder.FullBuilder(
		meta,
		[]libyaml.ConfigGroup{},
		map[string]interface{}{},
	)
	return *builder, err
}
