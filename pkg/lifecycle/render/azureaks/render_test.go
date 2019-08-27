package azureaks

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/test-mocks/inline"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRenderer(t *testing.T) {
	tests := []struct {
		name       string
		asset      api.AKSAsset
		kubeconfig string
	}{
		{
			name:       "empty",
			asset:      api.AKSAsset{},
			kubeconfig: "kubeconfig_",
		},
		{
			name: "named",
			asset: api.AKSAsset{
				ClusterName: "aClusterName",
			},
			kubeconfig: "kubeconfig_aClusterName",
		},
		{
			name: "named, custom path",
			asset: api.AKSAsset{
				ClusterName: "aClusterName",
				AssetShared: api.AssetShared{
					Dest: "aks.tf",
				},
			},
			kubeconfig: "kubeconfig_aClusterName",
		},
		{
			name: "named, in a directory",
			asset: api.AKSAsset{
				ClusterName: "aClusterName",
				AssetShared: api.AssetShared{
					Dest: "k8s/aks.tf",
				},
			},
			kubeconfig: "k8s/kubeconfig_aClusterName",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockInline := inline.NewMockRenderer(mc)
			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			bb := templates.NewBuilderBuilder(testLogger, v, &state.MockManager{})
			renderer := &LocalRenderer{
				Logger:         testLogger,
				BuilderBuilder: bb,
				Inline:         mockInline,
			}

			assetMatcher := &matchers.Is{
				Describe: "inline asset",
				Test: func(v interface{}) bool {
					_, ok := v.(api.InlineAsset)
					return ok
				},
			}

			rootFs := root.Fs{
				Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
				RootPath: "",
			}
			metadata := api.ReleaseMetadata{}
			groups := []libyaml.ConfigGroup{}
			templateContext := map[string]interface{}{}

			mockInline.EXPECT().Execute(
				rootFs,
				assetMatcher,
				metadata,
				templateContext,
				groups,
			).Return(func(ctx context.Context) error { return nil })

			err := renderer.Execute(
				rootFs,
				test.asset,
				metadata,
				templateContext,
				groups,
			)(context.Background())

			req.NoError(err)

			// test that the template function returns the correct kubeconfig path
			builder := templates.
				NewBuilderBuilder(log.NewNopLogger(), viper.New(), &state.MockManager{}).
				NewBuilder(
					&templates.ShipContext{},
				)

			aksTemplateFunc := `{{repl AzureAKS "%s" }}`
			kubeconfig, err := builder.String(fmt.Sprintf(aksTemplateFunc, test.asset.ClusterName))
			req.NoError(err)

			req.Equal(test.kubeconfig, kubeconfig, "Did not get expected kubeconfig path")

			otherKubeconfig, err := builder.String(fmt.Sprintf(aksTemplateFunc, "doesnotexist"))
			req.NoError(err)
			req.Empty(otherKubeconfig, "Expected path to nonexistent kubeconfig to be empty")
		})
	}
}

func TestRenderTerraformContents(t *testing.T) {
	var tests = []struct {
		name           string
		asset          api.AKSAsset
		kubeConfigPath string
		answer         string
	}{
		{
			name:           "With all optional values",
			kubeConfigPath: "kubeConfig_app",
			asset: api.AKSAsset{
				Azure: api.Azure{
					TenantID:               "tenant1",
					SubscriptionID:         "subscription1",
					ServicePrincipalID:     "serviceprincipal1",
					ServicePrincipalSecret: "serviceprincipalsecret",
					ResourceGroupName:      "ship",
					Location:               "US East",
				},
				ClusterName:       "app",
				KubernetesVersion: "v1.11.2",
				PublicKey:         "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDdEcdAqClaNZdHAGHhiSBobJo5ZUL3sDfrZbBQinLvx3HN/9UaXp5mimlzhUkUQwX4jPqJ78w52idmXItd55HVboSQ8uKaRicgLLaNhSqrNpb+W3k2RToRPsjuaCi6a8XET0kcma6NaIbae9n0+nKzTtadX/hkrPEMS56BYpnHjQ== user@example.com",
				NodeCount:         "2",
				NodeType:          "Standard_D1_v2",
				DiskGB:            "50",
			},
			answer: `
provider "azurerm" {
  tenant_id       = "tenant1"
  subscription_id = "subscription1"
  client_id       = "serviceprincipal1"
  client_secret   = "serviceprincipalsecret"
  version         = "~> 1.14"
}

resource "azurerm_resource_group" "ship" {
  name     = "ship"
  location = "US East"
}

resource "azurerm_kubernetes_cluster" "app" {
  name                = "app"
  location            = "US East"
  resource_group_name = "ship"
  dns_prefix          = "app"
  kubernetes_version  = "v1.11.2"

  linux_profile {
    admin_username = "admin"

    ssh_key {
      key_data = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDdEcdAqClaNZdHAGHhiSBobJo5ZUL3sDfrZbBQinLvx3HN/9UaXp5mimlzhUkUQwX4jPqJ78w52idmXItd55HVboSQ8uKaRicgLLaNhSqrNpb+W3k2RToRPsjuaCi6a8XET0kcma6NaIbae9n0+nKzTtadX/hkrPEMS56BYpnHjQ== user@example.com"
    }
  }

  agent_pool_profile {
    name            = "app"
    count           = 2
    vm_size         = "Standard_D1_v2"
    os_disk_size_gb = 50
  }

  service_principal {
    client_id     = "serviceprincipal1"
    client_secret = "serviceprincipalsecret"
  }
}

resource "local_file" "kubeconfig" {
  content = "${azurerm_kubernetes_cluster.app.kube_config_raw}"
  filename = "kubeConfig_app"
}
`,
		},
		{
			name:           "Without any optional values",
			kubeConfigPath: "kubeConfig_app",
			asset: api.AKSAsset{
				Azure: api.Azure{
					TenantID:               "tenant1",
					SubscriptionID:         "subscription1",
					ServicePrincipalID:     "serviceprincipal1",
					ServicePrincipalSecret: "serviceprincipalsecret",
					ResourceGroupName:      "ship",
					Location:               "US East",
				},
				ClusterName: "app",
				NodeCount:   "2",
				NodeType:    "Standard_D1_v2",
			},
			answer: `
provider "azurerm" {
  tenant_id       = "tenant1"
  subscription_id = "subscription1"
  client_id       = "serviceprincipal1"
  client_secret   = "serviceprincipalsecret"
  version         = "~> 1.14"
}

resource "azurerm_resource_group" "ship" {
  name     = "ship"
  location = "US East"
}

resource "azurerm_kubernetes_cluster" "app" {
  name                = "app"
  location            = "US East"
  resource_group_name = "ship"
  dns_prefix          = "app"

  agent_pool_profile {
    name            = "app"
    count           = 2
    vm_size         = "Standard_D1_v2"
  }

  service_principal {
    client_id     = "serviceprincipal1"
    client_secret = "serviceprincipalsecret"
  }
}

resource "local_file" "kubeconfig" {
  content = "${azurerm_kubernetes_cluster.app.kube_config_raw}"
  filename = "kubeConfig_app"
}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := renderTerraformContents(test.asset, test.kubeConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			if output != test.answer {
				t.Errorf("%s", output)
			}
		})
	}
}

func TestSafeClusterName(t *testing.T) {
	var tests = []struct {
		input  string
		answer string
	}{
		{
			input:  "My Cluster",
			answer: "mycluster",
		},
		{
			input:  "1 Cluster",
			answer: "cluster",
		},
		{
			input: "$$777	",
			answer: "y2x1c3rlciqk",
		},
		{
			input:  "!Apps",
			answer: "apps",
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			output := safeClusterName(test.input)
			if output != test.answer {
				t.Errorf("got %q, want %q", output, test.answer)
			}
		})
	}
}
