package amazonElasticKubernetesService

const newVPCTempl = `
variable "vpc_cidr" {
  type    = "string"
  default = "{{.CreatedVPC.VPCCIDR}}"
}

variable "vpc_public_subnets" {
  default = [{{range .CreatedVPC.PublicSubnets}}
    "{{.}}",{{end}}
  ]
}

variable "vpc_private_subnets" {
  default = [{{range .CreatedVPC.PrivateSubnets}}
    "{{.}}",{{end}}
  ]
}

variable "vpc_azs" {
  default = [{{range .CreatedVPC.Zones}}
    "{{.}}",{{end}}
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
`

const existingVPCTempl = `
locals {
  "eks_vpc"                 = "{{.ExistingVPC.VPCID}}"
  "eks_vpc_public_subnets"  = [{{range .ExistingVPC.PublicSubnets}}
    "{{.}}",{{end}}
  ]
  "eks_vpc_private_subnets" = [{{range .ExistingVPC.PrivateSubnets}}
    "{{.}}",{{end}}
  ]
}
`

const workerTempl = `
locals {
  "worker_group_count" = "{{len .AutoscalingGroups}}"
}

locals {
  "worker_groups" = [{{range .AutoscalingGroups}}
    {
      name                 = "{{.Name}}"
      asg_min_size         = "{{.GroupSize}}"
      asg_max_size         = "{{.GroupSize}}"
      asg_desired_capacity = "{{.GroupSize}}"
      instance_type        = "{{.MachineType}}"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },{{end}}
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "{{.Region}}"
}

variable "eks-cluster-name" {
  default = "{{.ClusterName}}"
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
