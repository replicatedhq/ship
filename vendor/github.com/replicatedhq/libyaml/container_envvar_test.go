package libyaml_test

import (
	"reflect"
	"testing"

	. "github.com/replicatedhq/libyaml"
	yaml "gopkg.in/yaml.v2"
)

func TestContainerEnvVar(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		expected ContainerEnvVar
		err      error
	}{
		{
			name: "with static_val",
			yaml: "name: ENV1\nstatic_val: static\nis_excluded_from_support: true\nwhen: true",
			expected: ContainerEnvVar{
				Name:                  "ENV1",
				Value:                 "static",
				StaticVal:             "static",
				IsExcludedFromSupport: "true",
				When: "true",
			},
		},
		{
			name: "with value",
			yaml: "name: ENV2\nvalue: static",
			expected: ContainerEnvVar{
				Name:      "ENV2",
				Value:     "static",
				StaticVal: "static",
			},
		},
		{
			name: "with no value",
			yaml: "name: ENV3",
			expected: ContainerEnvVar{
				Name: "ENV3",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var envVar ContainerEnvVar
			err := yaml.Unmarshal([]byte(c.yaml), &envVar)
			if c.err != nil {
				if err == nil {
					t.Error("Expecting error, got nil")
				}
			} else if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			} else if !reflect.DeepEqual(c.expected, envVar) {
				t.Errorf("Expecting %+v, got %+v", c.expected, envVar)
			}
		})
	}
}
