package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	yaml "gopkg.in/yaml.v2"
)

func TestSupport(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
components:
- name: DB
  containers:
  - source: public
    image_name: redis
    version: latest
support:
  files:
  - filename: /path/to/file
    source:
      component: DB
      container: redis
  commands:
  - filename: /path/to/command
    command: [ps, aux]
    source:
      component: DB
      container: redis
  timeout: 600
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err != nil {
			t.Error(err)
		}
		if root.Support.Timeout != "600" {
			t.Errorf("Support.Timeout == %s, want %s", root.Support.Timeout, "600")
		}
	})

	t.Run("valid flat", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
components:
- name: DB
  containers:
  - source: public
    image_name: redis
    version: latest
support:
  files:
  - filename: /path/to/file
    component: DB
    container: redis
  commands:
  - filename: /path/to/command
    command: [ps, aux]
    component: DB
    container: redis
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
components:
- name: DB
  containers:
  - source: public
    image_name: redis
    version: latest
support:
  files:
  - filename: /path/to/file
    source:
      component: DB
      container: notexists
  commands:
  - filename: /path/to/command
    command: [ps, aux]
    source:
      component: DB
      container: notexists
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Support.Files[0].Source.SourceContainerNative.Container":    "containerexists",
			"RootConfig.Support.Commands[0].Source.SourceContainerNative.Container": "containerexists",
		}); err != nil {
			t.Error(err)
		}
	})
}
