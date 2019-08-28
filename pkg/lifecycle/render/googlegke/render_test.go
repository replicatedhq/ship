package googlegke

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/pmezard/go-difflib/difflib"
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
		asset      api.GKEAsset
		kubeconfig string
	}{
		{
			name:       "empty",
			asset:      api.GKEAsset{},
			kubeconfig: "kubeconfig_",
		},
		{
			name: "named",
			asset: api.GKEAsset{
				ClusterName: "aClusterName",
			},
			kubeconfig: "kubeconfig_aClusterName",
		},
		{
			name: "named, custom path",
			asset: api.GKEAsset{
				ClusterName: "aClusterName",
				AssetShared: api.AssetShared{
					Dest: "gke.tf",
				},
			},
			kubeconfig: "kubeconfig_aClusterName",
		},
		{
			name: "named, in a directory",
			asset: api.GKEAsset{
				ClusterName: "aClusterName",
				AssetShared: api.AssetShared{
					Dest: "k8s/gke.tf",
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
			builder := getBuilder()

			gkeTemplateFunc := `{{repl GoogleGKE "%s" }}`
			kubeconfig, err := builder.String(fmt.Sprintf(gkeTemplateFunc, test.asset.ClusterName))
			req.NoError(err)

			req.Equal(test.kubeconfig, kubeconfig, "Did not get expected kubeconfig path")

			otherKubeconfig, err := builder.String(fmt.Sprintf(gkeTemplateFunc, "doesnotexist"))
			req.NoError(err)
			req.Empty(otherKubeconfig, "Expected path to nonexistent kubeconfig to be empty")
		})
	}
}

func getBuilder() templates.Builder {
	builderBuilder := templates.NewBuilderBuilder(log.NewNopLogger(), viper.New(), &state.MockManager{})

	builder := builderBuilder.NewBuilder(
		&templates.ShipContext{},
	)
	return builder
}

func TestRenderTerraformContents(t *testing.T) {
	tests := []struct {
		name     string
		asset    api.GKEAsset
		expected string
	}{
		{
			name: "simple",
			asset: api.GKEAsset{
				ClusterName: "simple-cluster",
			},
			expected: mustAsset("testassets/simple.tf"),
		},
		{
			name: "complex",
			asset: api.GKEAsset{
				GCPProvider: api.GCPProvider{
					Credentials: base64.StdEncoding.EncodeToString(
						[]byte("{\n  \"type\": \"service_account\",\n  \"project_id\": \"my-project\",\n  ...\n}"),
					),
					Project: "my-project",
					Region:  "us-east",
				},
				ClusterName:      "complex-cluster",
				Zone:             "us-east1-b",
				InitialNodeCount: "5",
				MachineType:      "n1-standard-4",
				AdditionalZones:  "us-east1-c,us-east1-d",
				MinMasterVersion: "1.10.6-gke.1",
			},
			expected: mustAsset("testassets/complex.tf"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := renderTerraformContents(test.asset)
			req.NoError(err)
			if actual != test.expected {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(test.expected),
					B:        difflib.SplitLines(actual),
					FromFile: "expected contents",
					ToFile:   "actual contents",
					Context:  3,
				}

				diffText, err := difflib.GetUnifiedDiffString(diff)
				req.NoError(err)

				t.Errorf("Test %s did not match, diff:\n%s", test.name, diffText)
			}
		})
	}
}

func mustAsset(name string) string {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestBuildAsset(t *testing.T) {
	type args struct {
		asset   api.GKEAsset
		builder *templates.Builder
	}
	tests := []struct {
		name    string
		args    args
		want    api.GKEAsset
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				asset: api.GKEAsset{
					ClusterName:      `{{repl "cluster_name_built"}}`,
					Zone:             `{{repl "zone_built"}}`,
					InitialNodeCount: `{{repl "initial_node_count_built"}}`,
					MachineType:      `{{repl "machine_type_built"}}`,
					AdditionalZones:  `{{repl "additional_zones_built"}}`,
					MinMasterVersion: `{{repl "min_master_version_built"}}`, // not built
				},
				builder: &templates.Builder{},
			},
			want: api.GKEAsset{
				ClusterName:      "cluster_name_built",
				Zone:             "zone_built",
				InitialNodeCount: "initial_node_count_built",
				MachineType:      "machine_type_built",
				AdditionalZones:  "additional_zones_built",
				MinMasterVersion: `{{repl "min_master_version_built"}}`, // not built
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, err := buildAsset(tt.args.asset, tt.args.builder)
			if !tt.wantErr {
				req.NoErrorf(err, "buildAsset() error = %v", err)
			} else {
				req.Error(err)
			}

			req.Equal(tt.want, got)
		})
	}
}
