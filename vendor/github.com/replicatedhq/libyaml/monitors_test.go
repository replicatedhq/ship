package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
)

func TestMonitorsDefault(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	runs := []ValidateTestRun{
		{
			`---
replicated_api_version: "2.8.0"
components:
  - name: wrong
    containers:
      - image_name: quay.io/getelk/logstash
        source: public
        version: "1.0.0"
monitors:
  cpuacct:
    - Logstash,quay.io/getelk/logstash
`,
			map[string]string{
				"RootConfig.Monitors.Cpuacct[0]": "componentexists",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
components:
  - name: Logstash
    containers:
      - image_name: quay.io/something/else
        source: public
        version: "1.0.0"
monitors:
  cpuacct:
    - Logstash,quay.io/getelk/logstash
`,
			map[string]string{
				"RootConfig.Monitors.Cpuacct[0]": "containerexists",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
monitors:
  cpuacct:
    - incomplete
`,
			map[string]string{
				"RootConfig.Monitors.Cpuacct[0]": "componentcontainer",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
monitors:
  cpuacct:
    - somethingswarm
swarm:
  minimum_node_count: "1"
`,
			map[string]string{
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
components:
  - name: wrong
    containers:
      - image_name: quay.io/getelk/logstash
        source: public
        version: "1.0.0"
monitors:
  memory:
    - Logstash,quay.io/getelk/logstash
`,
			map[string]string{
				"RootConfig.Monitors.Memory[0]": "componentexists",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
components:
  - name: Logstash
    containers:
      - image_name: quay.io/something/else
        source: public
        version: "1.0.0"
monitors:
  memory:
    - Logstash,quay.io/getelk/logstash
`,
			map[string]string{
				"RootConfig.Monitors.Memory[0]": "containerexists",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
monitors:
  memory:
    - incomplete
`,
			map[string]string{
				"RootConfig.Monitors.Memory[0]": "componentcontainer",
			},
		},
		{
			`---
replicated_api_version: "2.8.0"
monitors:
  memory:
    - somethingswarm
swarm:
  minimum_node_count: "1"
`,
			map[string]string{
			},
		},
	}

	RunValidateTest(t, runs, v, RootConfig{})
}
