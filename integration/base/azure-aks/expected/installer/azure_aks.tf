
provider "azurerm" {
  tenant_id       = "affca702-704f-4a80-a90d-64e1381fb5f0"
  subscription_id = "e9570f00-dbe5-4d1d-aa67-0eb9463fd9ad"
  client_id       = "ecee78e4-203e-4d8c-9018-0fabd50f6141"
  client_secret   = "OTViZDRmMC00NzEzLTRhZjUtOGJiMy03MDEzYzg2NTBhMzkK"
  version         = "~> 1.14"
}

resource "azurerm_resource_group" "Default" {
  name     = "Default"
  location = "US East"
}

resource "azurerm_kubernetes_cluster" "aksqa" {
  name                = "aksqa"
  location            = "US East"
  resource_group_name = "Default"
  dns_prefix          = "aksqa"
  kubernetes_version  = "1.11.2"

  agent_pool_profile {
    name            = "aksqa"
    count           = 1
    vm_size         = "Standard_D1_v2"
  }

  service_principal {
    client_id     = "ecee78e4-203e-4d8c-9018-0fabd50f6141"
    client_secret = "OTViZDRmMC00NzEzLTRhZjUtOGJiMy03MDEzYzg2NTBhMzkK"
  }
}

resource "local_file" "kubeconfig" {
  content = "${azurerm_kubernetes_cluster.aksqa.kube_config_raw}"
  filename = "kubeconfig_aksqa"
}
