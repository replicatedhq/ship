package amazoneks

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset (that produces an EKS cluster) as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.EKSAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// LocalRenderer renders a terraform asset by writing generated terraform source code
type LocalRenderer struct {
	BuilderBuilder *templates.BuilderBuilder
	Fs             afero.Afero
	Inline         inline.Renderer
	Logger         log.Logger
}

var _ Renderer = &LocalRenderer{}

func NewRenderer(
	bb *templates.BuilderBuilder,
	fs afero.Afero,
	inline inline.Renderer,
	logger log.Logger,
) Renderer {
	return &LocalRenderer{
		BuilderBuilder: bb,
		Fs:             fs,
		Inline:         inline,
		Logger:         logger,
	}
}

func (r *LocalRenderer) Execute(
	rootFs root.Fs,
	asset api.EKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		builder, err := r.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "init builder")
		}

		asset, err = buildAsset(asset, builder)
		if err != nil {
			return errors.Wrap(err, "build asset")
		}

		contents, err := renderTerraformContents(asset)
		if err != nil {
			return errors.Wrap(err, "render tf config")
		}

		assetsPath := "amazon_eks.tf"
		if asset.Dest != "" {
			assetsPath = asset.Dest
		}

		// save the path to the kubeconfig that running the generated terraform will produce
		templates.AddAmazonEKSPath(asset.ClusterName,
			path.Join(path.Dir(assetsPath), "kubeconfig_"+asset.ClusterName))

		// write the inline spec
		err = r.Inline.Execute(
			rootFs,
			api.InlineAsset{
				Contents: contents,
				AssetShared: api.AssetShared{
					Dest: assetsPath,
					Mode: asset.Mode,
				},
			},
			meta,
			templateContext,
			configGroups,
		)(ctx)

		if err != nil {
			return errors.Wrap(err, "write tf config")
		}
		return nil
	}
}

func buildAsset(asset api.EKSAsset, builder *templates.Builder) (api.EKSAsset, error) {
	var err error

	asset.ClusterName, err = builder.String(asset.ClusterName)
	if err != nil {
		return asset, errors.Wrap(err, "build cluster_name")
	}
	asset.Region, err = builder.String(asset.Region)
	if err != nil {
		return asset, errors.Wrap(err, "build region")
	}

	// build created vpc
	if asset.CreatedVPC != nil {
		asset.CreatedVPC.VPCCIDR, err = builder.String(asset.CreatedVPC.VPCCIDR)
		if err != nil {
			return asset, errors.Wrap(err, "build vpc_cidr")
		}

		for idx, zone := range asset.CreatedVPC.Zones {
			asset.CreatedVPC.Zones[idx], err = builder.String(zone)
			if err != nil {
				return asset, errors.Wrap(err, fmt.Sprintf("build vpc zone %d", idx))
			}
		}
		for idx, subnet := range asset.CreatedVPC.PublicSubnets {
			asset.CreatedVPC.PublicSubnets[idx], err = builder.String(subnet)
			if err != nil {
				return asset, errors.Wrap(err, fmt.Sprintf("build vpc public subnet %d", idx))
			}
		}
		for idx, subnet := range asset.CreatedVPC.PrivateSubnets {
			asset.CreatedVPC.PrivateSubnets[idx], err = builder.String(subnet)
			if err != nil {
				return asset, errors.Wrap(err, fmt.Sprintf("build vpc private subnet zone %d", idx))
			}
		}
	}

	// build existing vpc
	if asset.ExistingVPC != nil {
		asset.ExistingVPC.VPCID, err = builder.String(asset.ExistingVPC.VPCID)
		if err != nil {
			return asset, errors.Wrap(err, "build vpc_id")
		}

		for idx, subnet := range asset.ExistingVPC.PublicSubnets {
			asset.ExistingVPC.PublicSubnets[idx], err = builder.String(subnet)
			if err != nil {
				return asset, errors.Wrap(err, fmt.Sprintf("build vpc public subnet %d", idx))
			}
		}
		for idx, subnet := range asset.ExistingVPC.PrivateSubnets {
			asset.ExistingVPC.PrivateSubnets[idx], err = builder.String(subnet)
			if err != nil {
				return asset, errors.Wrap(err, fmt.Sprintf("build vpc private subnet zone %d", idx))
			}
		}
	}

	// build autoscaling groups
	for idx, group := range asset.AutoscalingGroups {
		asset.AutoscalingGroups[idx].Name, err = builder.String(group.Name)
		if err != nil {
			return asset, errors.Wrap(err, fmt.Sprintf("build autoscaling group %d name", idx))
		}
		asset.AutoscalingGroups[idx].GroupSize, err = builder.String(group.GroupSize)
		if err != nil {
			return asset, errors.Wrap(err, fmt.Sprintf("build autoscaling group %d group_size", idx))
		}
		asset.AutoscalingGroups[idx].MachineType, err = builder.String(group.MachineType)
		if err != nil {
			return asset, errors.Wrap(err, fmt.Sprintf("build autoscaling group %d machine_type", idx))
		}
	}

	return asset, nil
}

func renderTerraformContents(asset api.EKSAsset) (string, error) {
	templateString := ""
	if asset.CreatedVPC != nil {
		templateString = newVPCTempl
	} else if asset.ExistingVPC != nil {
		templateString = existingVPCTempl
	} else {
		return "", errors.New("a created or existing VPC must be provided")
	}

	templateString += workerTempl
	t, err := template.New("eksTemplate").Parse(templateString)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, asset)
}

func executeTemplate(t *template.Template, asset api.EKSAsset) (string, error) {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, asset); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
