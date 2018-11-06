package googlegke

const providerTempl = `
provider "google" {
  {{if .Credentials -}}
  credentials = <<EOF
{{.Credentials | b64dec}}
EOF
  {{end -}}
  project     = "{{.Project}}"
  region      = "{{.Region}}"
}
`

const clusterTempl = `
variable "cluster_name" {
  default = "{{.ClusterName}}"
}

variable "zone" {
  default = "{{.Zone}}"
}

variable "initial_node_count" {
  default = "{{if .InitialNodeCount -}}
{{.InitialNodeCount}}
{{- else -}}
3
{{- end}}"
}

variable "machine_type" {
  default = "{{if .MachineType -}}
{{.MachineType}}
{{- else -}}
n1-standard-1
{{- end}}"
}

variable "additional_zones" {
  type    = "list"
  default = [{{if .AdditionalZones}}{{range (split "," .AdditionalZones)}}
    "{{.}}",{{end}}
  {{end}}]
}

locals {
  min_master_version = "{{.MinMasterVersion}}"
}

resource "google_container_cluster" "{{.ClusterName}}" {
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

data "template_file" "kubeconfig_{{.ClusterName}}" {
  template = <<EOF
{{.KubeConfigTmpl}}
EOF

  vars {
    endpoint        = "https://${google_container_cluster.{{.ClusterName}}.endpoint}"
    cluster_auth    = "${google_container_cluster.{{.ClusterName}}.master_auth.0.cluster_ca_certificate}"
    kubeconfig_name = "{{.ClusterName}}"
    client_cert     = "${google_container_cluster.{{.ClusterName}}.master_auth.0.client_certificate}"
    client_key      = "${google_container_cluster.{{.ClusterName}}.master_auth.0.client_key}"
  }
}

resource "local_file" "kubeconfig_{{.ClusterName}}" {
  content = "${data.template_file.kubeconfig_{{.ClusterName}}.rendered}"
  filename = "kubeconfig_{{.ClusterName}}"
}
`

const kubeConfigTmpl = `
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
`
