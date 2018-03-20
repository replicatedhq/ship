package libyaml_test

import (
	"fmt"
	"strings"
	"testing"

	"reflect"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	yaml "gopkg.in/yaml.v2"
)

func TestAdminCommandSourceReplicated(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	// all three configs are equivalent (sort of)
	configs := [][]string{
		// test deprecated
		{
			`---
replicated_api_version: "1.3.2"
components:
- name: MyComponent
  containers:
  - source: public
    image_name: redis
    version: latest
admin_commands:
- alias: redis-sadd
  command: [redis-cli, sadd]
  run_type: exec
  component: MyComponent
  image:
    image_name: redis
    version: latest
`,
			"latest",
		},
		{
			// test indented
			`---
replicated_api_version: "1.3.2"
components:
- name: MyComponent
  containers:
  - source: public
    image_name: redis
    version: latest
admin_commands:
- alias: redis-sadd
  command: [redis-cli, sadd]
  run_type: exec
  source:
    replicated:
      component: MyComponent
      container: redis
`,
			"",
		},
		{
			// test non-indented
			`---
replicated_api_version: "1.3.2"
components:
- name: MyComponent
  containers:
  - source: public
    image_name: redis
    version: latest
admin_commands:
- alias: redis-sadd
  command: [redis-cli, sadd]
  run_type: exec
  component: MyComponent
  container: redis
`,
			"",
		},
	}
	for _, m := range configs {
		config := m[0]
		version := m[1]
		func(t *testing.T) {
			var root RootConfig
			err := yaml.Unmarshal([]byte(config), &root)
			if err != nil {
				t.Error(err)
				return
			}
			err = v.Struct(&root)
			if err != nil {
				t.Error(err)
			}

			expected := &AdminCommand{
				AdminCommandV2: AdminCommandV2{
					Alias:   "redis-sadd",
					Command: []string{"redis-cli", "sadd"},
					RunType: AdminCommandRunTypeExec,
					Source: SchedulerContainerSource{
						SourceContainerNative: &SourceContainerNative{
							Component: "MyComponent",
							Container: "redis",
						},
					},
				},
				AdminCommandV1: AdminCommandV1{
					Component: "MyComponent",
					Image: &CommandImage{
						Name:    "redis",
						Version: version, // v2 yaml omits container version
					},
				},
			}

			if len(root.AdminCommands) != 1 {
				t.Error("Expecting one admin command, got", len(root.AdminCommands))
				return
			}
			if !reflect.DeepEqual(expected, root.AdminCommands[0]) {
				t.Errorf("expected:\n%+#v\nactual:\n%+#v", expected, root.AdminCommands[0])
			}

			b, err := yaml.Marshal(root.AdminCommands[0])
			if err != nil {
				t.Fatal(err)
			}

			versionOut := version
			if versionOut == "" {
				versionOut = `""`
			}
			expectedOut := fmt.Sprintf(`alias: redis-sadd
command: [redis-cli, sadd]
run_type: exec
source:
  replicated:
    component: MyComponent
    container: redis
component: MyComponent
image:
  image_name: redis
  version: %s`, versionOut) // v2 yaml omits container version
			if strings.TrimSpace(expectedOut) != strings.TrimSpace(string(b)) {
				t.Errorf("expected:\n%s\n\nactual:\n%s", expectedOut, string(b))
			}
		}(t)
	}
}

func TestAdminCommandSourceKubernetes(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	// all three configs are equivalent (sort of)
	configs := []string{
		// test indented
		`---
replicated_api_version: "1.3.2"
components:
- name: MyComponent
  containers:
  - source: public
    image_name: redis
    version: latest
admin_commands:
- alias: redis-sadd
  command: [redis-cli, sadd]
  run_type: exec
  source:
    kubernetes:
      selector:
        app: redis
        role: master
      container: master
`,
		// test non-indented
		`---
replicated_api_version: "1.3.2"
components:
- name: MyComponent
  containers:
  - source: public
    image_name: redis
    version: latest
admin_commands:
- alias: redis-sadd
  command: [redis-cli, sadd]
  run_type: exec
  selector:
    app: redis
    role: master
  container: master
`,
	}
	for _, config := range configs {
		func(t *testing.T) {
			var root RootConfig
			err := yaml.Unmarshal([]byte(config), &root)
			if err != nil {
				t.Error(err)
				return
			}
			err = v.Struct(&root)
			if err != nil {
				t.Error(err)
			}

			expected := &AdminCommand{
				AdminCommandV2: AdminCommandV2{
					Alias:   "redis-sadd",
					Command: []string{"redis-cli", "sadd"},
					RunType: AdminCommandRunTypeExec,
					Source: SchedulerContainerSource{
						SourceContainerK8s: &SourceContainerK8s{
							Selector: map[string]string{
								"app":  "redis",
								"role": "master",
							},
							Selectors: map[string]string{
								"app":  "redis",
								"role": "master",
							},
							Container: "master",
						},
					},
				},
			}

			if len(root.AdminCommands) != 1 {
				t.Error("Expecting one admin command, got", len(root.AdminCommands))
				return
			}
			if !reflect.DeepEqual(expected, root.AdminCommands[0]) {
				t.Errorf("expected:\n%+#v\nactual:\n%+#v", expected, root.AdminCommands[0])
			}

			b, err := yaml.Marshal(root.AdminCommands[0])
			if err != nil {
				t.Fatal(err)
			}

			expectedOut := fmt.Sprintf(`alias: redis-sadd
command: [redis-cli, sadd]
run_type: exec
source:
  kubernetes:
    selector:
      app: redis
      role: master
    selectors:
      app: redis
      role: master
    container: master`) // v2 yaml omits container version
			if strings.TrimSpace(expectedOut) != strings.TrimSpace(string(b)) {
				t.Errorf("expected:\n%s\n\nactual:\n%s", expectedOut, string(b))
			}
		}(t)
	}
}
