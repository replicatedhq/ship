
variable "vpc_cidr" {
  type    = "string"
  default = "10.0.0.0/16"
}

variable "vpc_public_subnets" {
  default = [
    "10.0.1.0/24",
    "10.0.2.0/24",
  ]
}

variable "vpc_private_subnets" {
  default = [
    "10.0.129.0/24",
    "10.0.130.0/24",
  ]
}

variable "vpc_azs" {
  default = [
    "us-west-2a",
    "us-west-2b",
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
      name                 = "alpha"
      asg_min_size         = "3"
      asg_max_size         = "3"
      asg_desired_capacity = "3"
      instance_type        = "m5.2xlarge"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
    {
      name                 = "bravo"
      asg_min_size         = "1"
      asg_max_size         = "1"
      asg_desired_capacity = "1"
      instance_type        = "m5.4xlarge"

      subnets = "${join(",", local.eks_vpc_private_subnets)}"
    },
  ]
}

provider "aws" {
  version = "~> 1.27"
  region  = "us-west-2"
}

variable "eks-cluster-name" {
  default = "new-vpc-cluster"
  type    = "string"
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "1.4.0"

  cluster_name = "${var.eks-cluster-name}"

  subnets = ["${local.eks_vpc_private_subnets}", "${local.eks_vpc_public_subnets}"]

  vpc_id = "${local.eks_vpc}"

  worker_group_count = "${local.worker_group_count}"
  worker_groups      = "${local.worker_groups}"
}
