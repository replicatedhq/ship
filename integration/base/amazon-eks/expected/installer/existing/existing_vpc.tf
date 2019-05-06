
locals {
  "eks_vpc"                 = "abc123"
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
  version = "~> 2.7.0"
  region  = "us-east-1"
}

variable "eks-cluster-name" {
  default = "existing-vpc-cluster"
  type    = "string"
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "3.0.0"

  cluster_name = "${var.eks-cluster-name}"

  subnets = ["${local.eks_vpc_private_subnets}", "${local.eks_vpc_public_subnets}"]

  vpc_id = "${local.eks_vpc}"

  worker_group_count = "${local.worker_group_count}"
  worker_groups      = "${local.worker_groups}"
}
