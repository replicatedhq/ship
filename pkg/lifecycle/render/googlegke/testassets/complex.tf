
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

data "template_file" "kubeconfig_complex-cluster" {
  template = <<EOF

apiVersion: v1
preferences: {}
kind: Config

clusters:
- cluster:
    server: $${endpoint}
    certificate-authority-data: $${cluster_auth}
  name: $${kubeconfig_name}

contexts:
- context:
    cluster: $${kubeconfig_name}
    user: $${kubeconfig_name}
  name: $${kubeconfig_name}

current-context: $${kubeconfig_name}

users:
- name: $${kubeconfig_name}
  user:
    client-certificate-data: $${client_cert}
    client-key-data: $${client_key}

EOF

  vars {
    endpoint        = "https://${google_container_cluster.complex-cluster.endpoint}"
    cluster_auth    = "${google_container_cluster.complex-cluster.master_auth.0.cluster_ca_certificate}"
    kubeconfig_name = "complex-cluster"
    client_cert     = "${google_container_cluster.complex-cluster.master_auth.0.client_certificate}"
    client_key      = "${google_container_cluster.complex-cluster.master_auth.0.client_key}"
  }
}

resource "local_file" "kubeconfig_complex-cluster" {
  content = "${data.template_file.kubeconfig_complex-cluster.rendered}"
  filename = "kubeconfig_complex-cluster"
}
