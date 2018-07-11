package terraform

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
)

type Terraformer interface {
	Execute(ctx context.Context, release api.Release, step api.Terraform) error
	WithDaemon(d config.Daemon) Terraformer
}

type VendorTerraformer struct {
	Logger log.Logger
	Daemon config.Daemon
}

func NewTerraformer(
	logger log.Logger,
	daemon config.Daemon,
) Terraformer {
	return &VendorTerraformer{
		Logger: logger,
		Daemon: daemon,
	}
}

func (t *VendorTerraformer) WithDaemon(daemon config.Daemon) Terraformer {
	return &VendorTerraformer{
		Logger: t.Logger,
		Daemon: daemon,
	}
}

func (t *VendorTerraformer) Execute(ctx context.Context, release api.Release, step api.Terraform) error {
	panic("implement me")
}
