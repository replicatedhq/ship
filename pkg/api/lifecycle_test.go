package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestDeserializeLifecycle(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		expect Step
	}{
		{
			name: "message",
			yaml: `
---
lifecycle:
  v1:
    - message:
        contents: hi there
        level: warn`,

			expect: Step{
				Message: &Message{
					Contents: "hi there",
					Level:    "warn",
				},
			},
		},
		{
			name: "render",
			yaml: `
---
lifecycle:
  v1:
    - render: {}`,

			expect: Step{
				Render: &Render{},
			},
		},
		{
			name: "render with override assets",
			yaml: `
---
lifecycle:
  v1:
    - render: 
         root: ./bigapp
         assets:
           v1:
             - inline: { contents: fake }`,

			expect: Step{
				Render: &Render{
					Root: "./bigapp",
					Assets: &Assets{
						V1: []Asset{
							{
								Inline: &InlineAsset{
									Contents: "fake",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "terraform",
			yaml: `
---
lifecycle:
  v1:
    - terraform: {}`,

			expect: Step{
				Terraform: &Terraform{},
			},
		},
		{
			name: "kustomize minimal",

			yaml: `
---
lifecycle:
  v1:
    - kustomize:
         base: "k8s/"`,
			expect: Step{
				Kustomize: &Kustomize{
					Base: "k8s/",
				},
			},
		},
		{
			name: "kustomize with dest",

			yaml: `
---
lifecycle:
  v1:
    - kustomize:
         base: k8s/
         dest: rendered.yaml
         overlay: overlays/ship`,

			expect: Step{
				Kustomize: &Kustomize{
					Base:    "k8s/",
					Dest:    "rendered.yaml",
					Overlay: "overlays/ship",
				},
			},
		},
		{
			name: "helmIntro",
			yaml: `
---
lifecycle:
  v1:
    - helmIntro: {}`,
			expect: Step{
				HelmIntro: &HelmIntro{},
			},
		},
		{
			name: "helmValues",
			yaml: `
---
lifecycle:
  v1:
    - helmValues: {}`,
			expect: Step{
				HelmValues: &HelmValues{},
			},
		},
		{
			name: "requires",
			yaml: `
---
lifecycle:
  v1:
    - helmValues:
        id: values
        requires: 
          - intro
        invalidates: 
          - render `,
			expect: Step{
				HelmValues: &HelmValues{
					StepShared: StepShared{
						ID: "values",
						Requires: []string{
							"intro",
						},
						Invalidates: []string{
							"render",
						},
					},
				},
			},
		},
		{
			name: "helm-values-readme",
			yaml: `
---
lifecycle:
  v1:
    - helmValues:
        path: ./consul/values.yaml
        readme:
          contents: super dope
`,
			expect: Step{
				HelmValues: &HelmValues{
					Path: "./consul/values.yaml",
					Readme: &HelmValuesReadmeSource{
						Contents: "super dope",
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
			req.Equal(test.expect, spec.Lifecycle.V1[0])
		})
	}
}
