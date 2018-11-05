
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

resource "google_container_cluster" "complex-cluster" {
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
  content = "${google_container_cluster.complex-cluster.master_auth.0.client_certificate}"
  filename = "client_certificate"
}

resource "local_file" "client_key" {
  content = "${google_container_cluster.complex-cluster.master_auth.0.client_key}"
  filename = "client_key"
}

resource "local_file" "cluster_ca_certificate" {
  content = "${google_container_cluster.complex-cluster.master_auth.0.cluster_ca_certificate}"
  filename = "cluster_ca_certificate"
}

resource "local_file" "endpoint" {
  content = "${google_container_cluster.complex-cluster.endpoint}"
  filename = "endpoint"
}
