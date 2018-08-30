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

resource "google_container_cluster" "ship-{{.ClusterName}}" {
  name               = "${var.cluster_name}"
  zone               = "${var.zone}"
  initial_node_count = "${var.initial_node_count}"

  additional_zones = "${var.additional_zones}"

  min_master_version = "${local.min_master_version}"

  node_config {
    machine_type = "${var.machine_type}"
  }
}
`
