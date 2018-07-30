package amazonElasticKubernetesService

import (
	"context"
	"path"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.EKSAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// a LocalRenderer renders a terraform asset by vendoring in terraform source code
type LocalRenderer struct {
	Logger log.Logger
	Inline inline.Renderer
	Fs     afero.Afero
}

var _ Renderer = &LocalRenderer{}

func NewRenderer(
	logger log.Logger,
	inline inline.Renderer,
	fs afero.Afero,
) Renderer {
	return &LocalRenderer{
		Logger: logger,
		Inline: inline,
		Fs:     fs,
	}
}

func (r *LocalRenderer) Execute(
	asset api.EKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		contents, err := renderTerraformContents(asset)
		if err != nil {
			return errors.Wrap(err, "write tf config")
		}

		var assetsPath string
		if asset.Dest != "" {
			assetsPath = path.Join("terraform", asset.Dest)
		} else {
			assetsPath = path.Join("terraform", "amazon_elastic_kubernetes_service.tf")
		}

		// write the inline spec
		err = r.Inline.Execute(
			api.InlineAsset{
				Contents: contents,
				AssetShared: api.AssetShared{
					Dest: assetsPath,
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

func renderTerraformContents(asset api.EKSAsset) (string, error) {
	asgsString := renderASGs(asset.AutoscalingGroups)

	vpcsString := ""
	if asset.CreatedVPC != nil {
		vpcsString = renderNewVPC(*asset.CreatedVPC)
	} else if asset.ExistingVPC != nil {
		vpcsString = renderExistingVPC(*asset.ExistingVPC)
	} else {
		return "", errors.New("a VPC must be provided")
	}

	eksString := renderEKS(asset.Region, asset.ClusterName)

	return vpcsString + asgsString + eksString, nil
}

const itemsTempl = `
variable "%s" {
  default = [%s
  ]
}
`

const localItemsTempl = `
locals {
  "%s" = [%s
  ]
}
`

const itemTempl = `
    "%s",`

const localItemTempl = `
locals {
  "%s" = "%d"
}
`

const workerGroupTempl = `
    {
      name                 = "%s"
      asg_min_size         = "%d"
      asg_max_size         = "%d"
      asg_desired_capacity = "%d"
      instance_type        = "%s"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },`

func renderASGs(groups []api.EKSAutoscalingGroup) string {
	rendered := ""
	rendered += fmt.Sprintf(localItemTempl, "worker_group_count", len(groups))

	workerGroups := ""
	for _, group := range groups {
		workerGroups += fmt.Sprintf(workerGroupTempl, group.Name, group.GroupSize, group.GroupSize, group.GroupSize, group.MachineType)
	}

	rendered += fmt.Sprintf(localItemsTempl, "worker_groups", workerGroups)

	return rendered
}

const vpcCIDRTempl = `
variable "vpc_cidr" {
  type    = "string"
  default = "%s"
}
`

const vpcTempl = `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "1.37.0"
  name    = "eks-vpc"
  cidr    = "${var.vpc_cidr}"
  azs     = "${var.vpc_azs}"

  private_subnets = "${var.vpc_private_subnets}"
  public_subnets  = "${var.vpc_public_subnets}"

  map_public_ip_on_launch = true
  enable_nat_gateway      = true
  single_nat_gateway      = true

  tags = "${map("kubernetes.io/cluster/${var.eks-cluster-name}", "shared")}"
}
`

const vpcOutput = `
locals {
  "eks_vpc"                 = %s
  "eks_vpc_public_subnets"  = %s
  "eks_vpc_private_subnets" = %s
}
`

func renderNewVPC(vpc api.EKSCreatedVPC) string {
	rendered := ""
	rendered += fmt.Sprintf(vpcCIDRTempl, vpc.VPCCIDR)

	publicSubnets := ""
	for _, subnet := range vpc.PublicSubnets {
		publicSubnets += fmt.Sprintf(itemTempl, subnet)
	}
	rendered += fmt.Sprintf(itemsTempl, "vpc_public_subnets", publicSubnets)

	privateSubnets := ""
	for _, subnet := range vpc.PrivateSubnets {
		privateSubnets += fmt.Sprintf(itemTempl, subnet)
	}
	rendered += fmt.Sprintf(itemsTempl, "vpc_private_subnets", privateSubnets)

	AZs := ""
	for _, az := range vpc.Zones {
		AZs += fmt.Sprintf(itemTempl, az)
	}
	rendered += fmt.Sprintf(itemsTempl, "vpc_azs", AZs)

	rendered += vpcTempl

	rendered += fmt.Sprintf(vpcOutput, `"${module.vpc.vpc_id}"`, `"${module.vpc.public_subnets}"`, `"${module.vpc.private_subnets}"`)

	return rendered
}

const listTempl = `[%s
  ]`

func renderExistingVPC(vpc api.EKSExistingVPC) string {
	rendered := ""

	publicIDs := ""
	for _, publicID := range vpc.PublicSubnets {
		publicIDs += fmt.Sprintf(itemTempl, publicID)
	}
	publicIDs = fmt.Sprintf(listTempl, publicIDs)

	privateIDs := ""
	for _, privateID := range vpc.PrivateSubnets {
		privateIDs += fmt.Sprintf(itemTempl, privateID)
	}
	privateIDs = fmt.Sprintf(listTempl, privateIDs)

	vpcID := fmt.Sprintf(`"%s"`, vpc.VPCID)

	rendered += fmt.Sprintf(vpcOutput, vpcID, publicIDs, privateIDs)

	return rendered
}

const eksTempl = `
provider "aws" {
  version = "~> 1.27"
  region  = "%s"
}

variable "eks-cluster-name" {
  default = "%s"
  type    = "string"
}

module "eks" {
  #source = "terraform-aws-modules/eks/aws"
  source  = "laverya/eks/aws"
  version = "1.4.0"

  cluster_name = "${var.eks-cluster-name}"

  subnets = ["${local.eks_vpc_private_subnets}", "${local.eks_vpc_public_subnets}"]

  vpc_id = "${local.eks_vpc}"

  worker_group_count = "${local.worker_group_count}"
  worker_groups      = "${local.worker_groups}"
}
`

func renderEKS(region, name string) string {
	return fmt.Sprintf(eksTempl, region, name)
}
