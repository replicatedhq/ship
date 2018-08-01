
locals {
  "eks_vpc"                 = "abc123"
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
