
provider "google" {
  credentials = <<EOF
{
  "type": "service_account",
  "project_id": "my-project",
  ...
}
EOF
  project     = "my-project"
  region      = "us-east"
}

variable "cluster_name" {
  default = "complex-cluster"
}

variable "zone" {
  default = "us-east1-b"
}

variable "initial_node_count" {
  default = "5"
}

variable "machine_type" {
  default = "n1-standard-4"
}

variable "additional_zones" {
  type    = "list"
  default = [
    "us-east1-c",
    "us-east1-d",
  ]
}

locals {
  min_master_version = "1.10.6-gke.1"
}

resource "google_container_cluster" "ship-complex-cluster" {
  name               = "${var.cluster_name}"
  zone               = "${var.zone}"
  initial_node_count = "${var.initial_node_count}"

  additional_zones = "${var.additional_zones}"

  min_master_version = "${local.min_master_version}"

  node_config {
    machine_type = "${var.machine_type}"
  }
}
