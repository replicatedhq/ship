package terraform

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
)

// Renderer is something that can render a terraform asset as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.TerraformAsset,
		meta api.ReleaseMetadata,
		configGroups []libyaml.ConfigGroup,
		templateContext map[string]interface{},
	) func(ctx context.Context) error
}

// a VendorRenderer renders a terraform asset by vendoring in terraform source code
type VendorRenderer struct {
	Logger log.Logger
}

var _ Renderer = &VendorRenderer{}

func NewRenderer(logger log.Logger) Renderer {
	return &VendorRenderer{
		Logger: logger,
	}
}

func (r *VendorRenderer) Execute(
	asset api.TerraformAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return nil
	}
}
