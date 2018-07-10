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
