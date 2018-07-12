package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
)

func TestK8sRequirements(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	runs := []ValidateTestRun{
		ValidateTestRun{
			`---
replicated_api_version: "1.3.2"
kubernetes:
  requirements: {}
`,
			map[string]string{},
		},
		ValidateTestRun{
			`---
replicated_api_version: "1.3.2"
kubernetes:
  requirements:
    server_version: ">=1.5.0"
    api_versions:
    - extensions/v1beta1
    - ""
    cluster_size: 3
    total_cores: 6
    total_memory: 10KB
`,
			map[string]string{
				"RootConfig.K8s.Requirements.APIVersions[1]": "required",
			},
		},
		ValidateTestRun{
			`---
replicated_api_version: "1.3.2"
kubernetes:
  requirements:
    server_version: ">=1.5"
    cluster_size: -1
    total_cores: -1
    total_memory: 10blah
`,
			map[string]string{
				"RootConfig.K8s.Requirements.ServerVersion": "semverrange",
				"RootConfig.K8s.Requirements.ClusterSize":   "number",
				"RootConfig.K8s.Requirements.TotalCores":    "number",
				"RootConfig.K8s.Requirements.TotalMemory":   "bytes|quantity",
			},
		},
	}

	RunValidateTest(t, runs, v, RootConfig{})
}

func TestK8sPVClaims(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	runs := []ValidateTestRun{
		ValidateTestRun{
			`---
replicated_api_version: "1.3.2"
kubernetes:
  persistent_volume_claims:
  - name: pv1
    storage: 10GB
    access_modes: ["RWO"]
  - name: pv2
    storage: 10Gi
    access_modes: ["RWO"]
`,
			map[string]string{},
		},
		ValidateTestRun{
			`---
replicated_api_version: "1.3.2"
kubernetes:
  persistent_volume_claims:
  - name: ""
    storage: 10blah
`,
			map[string]string{
				"RootConfig.K8s.PVClaims[0].Name":    "required",
				"RootConfig.K8s.PVClaims[0].Storage": "bytes|quantity",
			},
		},
	}

	RunValidateTest(t, runs, v, RootConfig{})
}
