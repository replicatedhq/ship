package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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
			name: "kustomize",

			yaml: `
---
lifecycle:
  v1:
    - kustomize:
         base_path: "k8s/"`,

			expect: Step{
				Kustomize: &Kustomize{
					BasePath: "k8s/",
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
