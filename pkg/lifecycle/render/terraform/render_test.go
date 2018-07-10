package terraform

import (
	"context"
	"testing"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
)

func TestRenderer(t *testing.T) {
	tests := []struct {
		name  string
		asset api.TerraformAsset
	}{
		{
			name: "empty",
			asset: api.TerraformAsset{
				Inline: "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			renderer := &VendorRenderer{
				Logger: &logger.TestLogger{T: t},
			}

			err := renderer.Execute(
				test.asset,
				api.ReleaseMetadata{},
				[]libyaml.ConfigGroup{},
				map[string]interface{}{},
			)(context.Background())

			req.NoError(err)
		})
	}
}
