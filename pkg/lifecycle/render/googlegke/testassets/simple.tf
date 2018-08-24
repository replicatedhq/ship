
variable "cluster_name" {
  default = "simple-cluster"
}

variable "zone" {
  default = ""
}

variable "initial_node_count" {
  default = "3"
}

variable "machine_type" {
  default = "n1-standard-1"
}

variable "additional_zones" {
  type    = "list"
  default = []
}

locals {
  min_master_version = ""
}

resource "google_container_cluster" "ship-simple-cluster" {
  name               = "${var.cluster_name}"
  zone               = "${var.zone}"
  initial_node_count = "${var.initial_node_count}"

  additional_zones = "${var.additional_zones}"

  min_master_version = "${local.min_master_version}"

  node_config {
    machine_type = "${var.machine_type}"
  }
}
