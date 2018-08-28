package azureaks

const clusterTempl = `
provider "azurerm" {
  tenant_id       = "{{ .Azure.TenantID }}"
  subscription_id = "{{ .Azure.SubscriptionID }}"
  client_id       = "{{ .Azure.ServicePrincipalID }}"
  client_secret   = "{{ .Azure.ServicePrincipalSecret }}"
  version         = "~> 1.14"
}

resource "azurerm_resource_group" "{{ .Azure.ResourceGroupName }}" {
  name     = "{{ .Azure.ResourceGroupName }}"
  location = "{{ .Azure.Location }}"
}

resource "azurerm_kubernetes_cluster" "{{ .SafeClusterName }}" {
  name                = "{{ .ClusterName }}"
  location            = "{{ .Azure.Location }}"
  resource_group_name = "{{ .Azure.ResourceGroupName }}"
  dns_prefix          = "{{ .SafeClusterName }}"{{if .KubernetesVersion }}
  kubernetes_version  = "{{ .KubernetesVersion }}"
  {{- end}}
  {{- if .PublicKey }}

  linux_profile {
    admin_username = "admin"

    ssh_key {
      key_data = "{{ .PublicKey }}"
    }
  }
  {{- end}}

  agent_pool_profile {
    name            = "{{ .SafeClusterName }}"
    count           = {{ .NodeCount }}
    vm_size         = "{{ .NodeType }}"
    {{- if .DiskGB }}
    os_disk_size_gb = {{ .DiskGB }}
    {{- end }}
  }

  service_principal {
    client_id     = "{{ .Azure.ServicePrincipalID }}"
    client_secret = "{{ .Azure.ServicePrincipalSecret }}"
  }
}

resource "local_file" "kubeconfig" {
  content = "${azurerm_kubernetes_cluster.{{ .SafeClusterName }}.kube_config_raw}"
  filename = "{{ .KubeConfigPath }}"
}
`
