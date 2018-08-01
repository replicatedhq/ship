package amazonElasticKubernetesService

import (
	"context"
	"html/template"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/api/amazonElasticKubernetesService"
	"github.com/replicatedhq/ship/pkg/test-mocks/inline"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/stretchr/testify/require"
)

func TestRenderer(t *testing.T) {
	tests := []struct {
		name  string
		asset api.EKSAsset
	}{
		{
			name:  "empty",
			asset: api.EKSAsset{ExistingVPC: &amazonElasticKubernetesService.EKSExistingVPC{}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockInline := inline.NewMockRenderer(mc)

			renderer := &LocalRenderer{
				Logger: &logger.TestLogger{T: t},
				Inline: mockInline,
			}

			assetMatcher := &matchers.Is{
				Describe: "inline asset",
				Test: func(v interface{}) bool {
					_, ok := v.(api.InlineAsset)
					if !ok {
						return false
					}
					return true
				},
			}

			metadata := api.ReleaseMetadata{}
			groups := []libyaml.ConfigGroup{}
			templateContext := map[string]interface{}{}

			mockInline.EXPECT().Execute(
				assetMatcher,
				metadata,
				templateContext,
				groups,
			).Return(func(ctx context.Context) error { return nil })

			err := renderer.Execute(
				test.asset,
				metadata,
				templateContext,
				groups,
			)(context.Background())

			req.NoError(err)
		})
	}
}

// this function allows testing the worker template independently
func testRenderEKS(asset api.EKSAsset) (string, error) {
	t, err := template.New("asgsTemplate").Parse(workerTempl)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, asset)
}

func TestRenderEKSs(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		asset    api.EKSAsset
	}{
		{
			name:  "empty",
			asset: api.EKSAsset{},
			expected: `
locals {
  "worker_group_count" = "0"
}

locals {
  "worker_groups" = [
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = ""
}

variable "eks-cluster-name" {
  default = ""
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
`,
		},
		{
			name: "one",
			asset: api.EKSAsset{
				AutoscalingGroups: []amazonElasticKubernetesService.EKSAutoscalingGroup{
					{
						Name:        "onegroup",
						GroupSize:   3,
						MachineType: "m5.large",
					},
				},
				Region:      "test-region",
				ClusterName: "a-test-cluster",
			},
			expected: `
locals {
  "worker_group_count" = "1"
}

locals {
  "worker_groups" = [
    {
      name                 = "onegroup"
      asg_min_size         = "3"
      asg_max_size         = "3"
      asg_desired_capacity = "3"
      instance_type        = "m5.large"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "test-region"
}

variable "eks-cluster-name" {
  default = "a-test-cluster"
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
`,
		},
		{
			name: "two",
			asset: api.EKSAsset{
				AutoscalingGroups: []amazonElasticKubernetesService.EKSAutoscalingGroup{
					{
						Name:        "onegroup",
						GroupSize:   3,
						MachineType: "m5.large",
					},
					{
						Name:        "twogroup",
						GroupSize:   1,
						MachineType: "m5.xlarge",
					},
				},
				Region:      "double-test-region",
				ClusterName: "double-test-cluster",
			},
			expected: `
locals {
  "worker_group_count" = "2"
}

locals {
  "worker_groups" = [
    {
      name                 = "onegroup"
      asg_min_size         = "3"
      asg_max_size         = "3"
      asg_desired_capacity = "3"
      instance_type        = "m5.large"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
    {
      name                 = "twogroup"
      asg_min_size         = "1"
      asg_max_size         = "1"
      asg_desired_capacity = "1"
      instance_type        = "m5.xlarge"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "double-test-region"
}

variable "eks-cluster-name" {
  default = "double-test-cluster"
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
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := testRenderEKS(test.asset)
			req.NoError(err)
			if actual != test.expected {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(test.expected),
					B:        difflib.SplitLines(actual),
					FromFile: "expected contents",
					ToFile:   "actual contents",
					Context:  3,
				}

				diffText, err := difflib.GetUnifiedDiffString(diff)
				req.NoError(err)

				t.Errorf("Test %s did not match, diff:\n%s", test.name, diffText)
			}
		})
	}
}

// this function allows testing the new VPC template independently
func testRenderNewVPC(vpc api.EKSAsset) (string, error) {
	t, err := template.New("eksTemplate").Parse(newVPCTempl)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, vpc)
}

func TestRenderVPC(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		asset    api.EKSAsset
	}{
		{
			name: "empty",
			asset: api.EKSAsset{
				CreatedVPC: &amazonElasticKubernetesService.EKSCreatedVPC{},
			},
			expected: `
variable "vpc_cidr" {
  type    = "string"
  default = ""
}

variable "vpc_public_subnets" {
  default = [
  ]
}

variable "vpc_private_subnets" {
  default = [
  ]
}

variable "vpc_azs" {
  default = [
  ]
}

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

locals {
  "eks_vpc"                 = "${module.vpc.vpc_id}"
  "eks_vpc_public_subnets"  = "${module.vpc.public_subnets}"
  "eks_vpc_private_subnets" = "${module.vpc.private_subnets}"
}
`,
		},
		{
			name: "basic vpc",
			asset: api.EKSAsset{
				CreatedVPC: &amazonElasticKubernetesService.EKSCreatedVPC{
					VPCCIDR:        "10.0.0.0/16",
					PublicSubnets:  []string{"10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24", "10.0.4.0/24"},
					PrivateSubnets: []string{"10.128.1.0/24", "10.128.2.0/24", "10.128.3.0/24", "10.128.4.0/24"},
					Zones:          []string{"us-west-2a", "us-west-2b", "us-west-2c", "us-west-2d"},
				},
			},
			expected: `
variable "vpc_cidr" {
  type    = "string"
  default = "10.0.0.0/16"
}

variable "vpc_public_subnets" {
  default = [
    "10.0.1.0/24",
    "10.0.2.0/24",
    "10.0.3.0/24",
    "10.0.4.0/24",
  ]
}

variable "vpc_private_subnets" {
  default = [
    "10.128.1.0/24",
    "10.128.2.0/24",
    "10.128.3.0/24",
    "10.128.4.0/24",
  ]
}

variable "vpc_azs" {
  default = [
    "us-west-2a",
    "us-west-2b",
    "us-west-2c",
    "us-west-2d",
  ]
}

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

locals {
  "eks_vpc"                 = "${module.vpc.vpc_id}"
  "eks_vpc_public_subnets"  = "${module.vpc.public_subnets}"
  "eks_vpc_private_subnets" = "${module.vpc.private_subnets}"
}
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := testRenderNewVPC(test.asset)
			req.NoError(err)
			if actual != test.expected {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(test.expected),
					B:        difflib.SplitLines(actual),
					FromFile: "expected contents",
					ToFile:   "actual contents",
					Context:  3,
				}

				diffText, err := difflib.GetUnifiedDiffString(diff)
				req.NoError(err)

				t.Errorf("Test %s did not match, diff:\n%s", test.name, diffText)
			}
		})
	}
}

// this function allows testing the existing VPC template independently
func testRenderExistingVPC(vpc api.EKSAsset) (string, error) {
	t, err := template.New("eksTemplate").Parse(existingVPCTempl)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, vpc)
}

func TestRenderExistingVPC(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		asset    api.EKSAsset
	}{
		{
			name:  "empty",
			asset: api.EKSAsset{ExistingVPC: &amazonElasticKubernetesService.EKSExistingVPC{}},
			expected: `
locals {
  "eks_vpc"                 = ""
  "eks_vpc_public_subnets"  = [
  ]
  "eks_vpc_private_subnets" = [
  ]
}
`,
		},
		{
			name: "basic vpc",
			asset: api.EKSAsset{
				ExistingVPC: &amazonElasticKubernetesService.EKSExistingVPC{
					VPCID:          "vpcid",
					PublicSubnets:  []string{"abc123-a", "abc123-b"},
					PrivateSubnets: []string{"xyz789-a", "xyz789-b"},
				},
			},
			expected: `
locals {
  "eks_vpc"                 = "vpcid"
  "eks_vpc_public_subnets"  = [
    "abc123-a",
    "abc123-b",
  ]
  "eks_vpc_private_subnets" = [
    "xyz789-a",
    "xyz789-b",
  ]
}
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := testRenderExistingVPC(test.asset)
			req.NoError(err)
			if actual != test.expected {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(test.expected),
					B:        difflib.SplitLines(actual),
					FromFile: "expected contents",
					ToFile:   "actual contents",
					Context:  3,
				}

				diffText, err := difflib.GetUnifiedDiffString(diff)
				req.NoError(err)

				t.Errorf("Test %s did not match, diff:\n%s", test.name, diffText)
			}
		})
	}
}

func TestRenderTerraform(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		vpc      api.EKSAsset
	}{
		{
			name: "empty",
			vpc:  api.EKSAsset{ExistingVPC: &amazonElasticKubernetesService.EKSExistingVPC{}},
			expected: `
locals {
  "eks_vpc"                 = ""
  "eks_vpc_public_subnets"  = [
  ]
  "eks_vpc_private_subnets" = [
  ]
}

locals {
  "worker_group_count" = "0"
}

locals {
  "worker_groups" = [
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = ""
}

variable "eks-cluster-name" {
  default = ""
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
`,
		},
		{
			name: "existing vpc",
			vpc: api.EKSAsset{
				ClusterName: "existing-vpc-cluster",
				Region:      "us-east-1",
				ExistingVPC: &amazonElasticKubernetesService.EKSExistingVPC{
					VPCID:          "existing_vpcid",
					PublicSubnets:  []string{"abc123-a", "abc123-b"},
					PrivateSubnets: []string{"xyz789-a", "xyz789-b"},
				},
				AutoscalingGroups: []amazonElasticKubernetesService.EKSAutoscalingGroup{
					{
						Name:        "onegroup",
						GroupSize:   3,
						MachineType: "m5.large",
					},
				},
			},
			expected: `
locals {
  "eks_vpc"                 = "existing_vpcid"
  "eks_vpc_public_subnets"  = [
    "abc123-a",
    "abc123-b",
  ]
  "eks_vpc_private_subnets" = [
    "xyz789-a",
    "xyz789-b",
  ]
}

locals {
  "worker_group_count" = "1"
}

locals {
  "worker_groups" = [
    {
      name                 = "onegroup"
      asg_min_size         = "3"
      asg_max_size         = "3"
      asg_desired_capacity = "3"
      instance_type        = "m5.large"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "us-east-1"
}

variable "eks-cluster-name" {
  default = "existing-vpc-cluster"
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
`,
		},
		{
			name: "new vpc",
			vpc: api.EKSAsset{
				ClusterName: "new-vpc-cluster",
				Region:      "us-east-1",
				CreatedVPC: &amazonElasticKubernetesService.EKSCreatedVPC{
					VPCCIDR:        "10.0.0.0/16",
					PublicSubnets:  []string{"10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24", "10.0.4.0/24"},
					PrivateSubnets: []string{"10.128.1.0/24", "10.128.2.0/24", "10.128.3.0/24", "10.128.4.0/24"},
					Zones:          []string{"us-west-2a", "us-west-2b", "us-west-2c", "us-west-2d"},
				},
				AutoscalingGroups: []amazonElasticKubernetesService.EKSAutoscalingGroup{
					{
						Name:        "onegroup",
						GroupSize:   3,
						MachineType: "m5.large",
					},
					{
						Name:        "twogroup",
						GroupSize:   2,
						MachineType: "m4.large",
					},
				},
			},
			expected: `
variable "vpc_cidr" {
  type    = "string"
  default = "10.0.0.0/16"
}

variable "vpc_public_subnets" {
  default = [
    "10.0.1.0/24",
    "10.0.2.0/24",
    "10.0.3.0/24",
    "10.0.4.0/24",
  ]
}

variable "vpc_private_subnets" {
  default = [
    "10.128.1.0/24",
    "10.128.2.0/24",
    "10.128.3.0/24",
    "10.128.4.0/24",
  ]
}

variable "vpc_azs" {
  default = [
    "us-west-2a",
    "us-west-2b",
    "us-west-2c",
    "us-west-2d",
  ]
}

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

locals {
  "eks_vpc"                 = "${module.vpc.vpc_id}"
  "eks_vpc_public_subnets"  = "${module.vpc.public_subnets}"
  "eks_vpc_private_subnets" = "${module.vpc.private_subnets}"
}

locals {
  "worker_group_count" = "2"
}

locals {
  "worker_groups" = [
    {
      name                 = "onegroup"
      asg_min_size         = "3"
      asg_max_size         = "3"
      asg_desired_capacity = "3"
      instance_type        = "m5.large"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
    {
      name                 = "twogroup"
      asg_min_size         = "2"
      asg_max_size         = "2"
      asg_desired_capacity = "2"
      instance_type        = "m4.large"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "us-east-1"
}

variable "eks-cluster-name" {
  default = "new-vpc-cluster"
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
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := renderTerraformContents(test.vpc)
			req.NoError(err)
			if actual != test.expected {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(test.expected),
					B:        difflib.SplitLines(actual),
					FromFile: "expected contents",
					ToFile:   "actual contents",
					Context:  3,
				}

				diffText, err := difflib.GetUnifiedDiffString(diff)
				req.NoError(err)

				t.Errorf("Test %s did not match, diff:\n%s", test.name, diffText)
			}
		})
	}
}
