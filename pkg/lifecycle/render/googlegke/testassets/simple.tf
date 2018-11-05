
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

resource "google_container_cluster" "simple-cluster" {
  name               = "${var.cluster_name}"
  zone               = "${var.zone}"
  initial_node_count = "${var.initial_node_count}"

  additional_zones = "${var.additional_zones}"

  min_master_version = "${local.min_master_version}"

  node_config {
    machine_type = "${var.machine_type}"
  }

  enable_legacy_abac = "true"
}

resource "local_file" "client_certificate" {
  content = "${google_container_cluster.simple-cluster.master_auth.0.client_certificate}"
  filename = "client_certificate"
}

resource "local_file" "client_key" {
  content = "${google_container_cluster.simple-cluster.master_auth.0.client_key}"
  filename = "client_key"
}

resource "local_file" "cluster_ca_certificate" {
  content = "${google_container_cluster.simple-cluster.master_auth.0.cluster_ca_certificate}"
  filename = "cluster_ca_certificate"
}

resource "local_file" "endpoint" {
  content = "${google_container_cluster.simple-cluster.endpoint}"
  filename = "endpoint"
}
