package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestDeserialize(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		expect Asset
	}{
		{
			name: "layer",
			yaml: `
---
assets:
  v1:
    - dockerlayer:
        image: foo
        source: public
        dest: docker/layers/
        layer: abcdefg`,

			expect: Asset{
				DockerLayer: &DockerLayerAsset{
					DockerAsset: DockerAsset{
						Image:  "foo",
						Source: "public",
						AssetShared: AssetShared{
							Dest: "docker/layers/",
						},
					},
					Layer: "abcdefg",
				},
			},
		},
		{
			name: "docker",
			yaml: `
---
assets:
  v1:
    - docker:
        image: foo
        source: public
        dest: docker/foo.tar`,

			expect: Asset{
				Docker: &DockerAsset{
					Image:  "foo",
					Source: "public",
					AssetShared: AssetShared{
						Dest: "docker/foo.tar",
					},
				},
			},
		},
		{
			name: "inline",
			yaml: `
---
assets:
  v1:
    - inline:
        contents: hi
        dest: greetings/hello.txt`,

			expect: Asset{
				Inline: &InlineAsset{
					Contents: "hi",
					AssetShared: AssetShared{
						Dest: "greetings/hello.txt",
					},
				},
			},
		},
		{
			name: "github",
			yaml: `
---
assets:
  v1:
    - github:
        repo: replicatedhq/test_specs
        ref: refs/heads/master
        path: k8s/api/deployment.yml
        source: private
        dest: k8s/deployment.yml`,

			expect: Asset{
				GitHub: &GitHubAsset{
					Repo:   "replicatedhq/test_specs",
					Ref:    "refs/heads/master",
					Path:   "k8s/api/deployment.yml",
					Source: "private",
					AssetShared: AssetShared{
						Dest: "k8s/deployment.yml",
					},
				},
			},
		},
		{
			name: "helm",
			yaml: `
---
assets:
  v1:
    - helm:
        local:
           chart_root: helm/charts/src/nginx
        dest: helm/charts/rendered/`,

			expect: Asset{
				Helm: &HelmAsset{
					Local: &LocalHelmOpts{
						ChartRoot: "helm/charts/src/nginx",
					},
					AssetShared: AssetShared{
						Dest: "helm/charts/rendered/",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			var spec Spec

			err := yaml.Unmarshal([]byte(test.yaml), &spec)
			req.NoError(err)
			req.Equal(test.expect, spec.Assets.V1[0])
		})
	}
}
