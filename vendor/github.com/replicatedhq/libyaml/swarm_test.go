package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	yaml "gopkg.in/yaml.v2"
)

func TestSwarmMinNodeCount(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  minimum_node_count: 3
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

	t.Run("non-number", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  minimum_node_count: asd
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.MinNodeCount": "number",
		}); err != nil {
			t.Error(err)
		}
	})
}

func TestSwarmNode(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - role: manager
    labels:
      a: b
      c:
    minimum_count: 3
    host_requirements:
      docker_version: 17.03.1
      cpu_cores: 2
      cpu_mhz: 2000
      memory: 4GB
      disk_space: 10GB
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
}

func TestSwarmNodeRole(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - role: worker
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
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - role: blah
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Nodes[0].Role": "swarmnoderole",
		}); err != nil {
			t.Error(err)
		}
	})
}

func TestSwarmNodeLabels(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - labels:
      a: b
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
}

func TestSwarmNodeMinCount(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - minimum_count: 3
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

	t.Run("non-number", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  nodes:
  - minimum_count: asd
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Nodes[0].MinCount": "number",
		}); err != nil {
			t.Error(err)
		}
	})
}

func TestSwarmSecret(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	if err := RegisterValidations(v); err != nil {
		t.Fatal(err)
	}

	t.Run("valid with labels", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  secrets:
  - name: foo
    value: bar
    labels:
      foo: bar
      baz: boo
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

	t.Run("valid without labels", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  secrets:
  - name: foo
    value: bar
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

	t.Run("invalid no value", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  secrets:
  - name: foo
    value:
    labels:
      foo: bar
      baz: boo
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Secrets[0].Value": "required",
		}); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid no name", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  secrets:
  - name: 
    value: bar
    labels:
      foo: bar
      baz: boo
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Secrets[0].Name": "required",
		}); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid empty label keys", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  secrets:
  - name: foo
    value: bar
    labels:
      "": bar
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Secrets[0].Labels": "mapkeylengthnonzero",
		}); err != nil {
			t.Error(err)
		}
	})
}

func TestSwarmConfig(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	if err := RegisterValidations(v); err != nil {
		t.Fatal(err)
	}

	t.Run("valid with labels", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  configs:
  - name: foo
    value: bar
    labels:
      foo: bar
      baz: boo
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

	t.Run("valid without labels", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  configs:
  - name: foo
    value: bar
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

	t.Run("invalid no value", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  configs:
  - name: foo
    value:
    labels:
      foo: bar
      baz: boo
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Configs[0].Value": "required",
		}); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid no name", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  configs:
  - name: 
    value: bar
    labels:
      foo: bar
      baz: boo
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Configs[0].Name": "required",
		}); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid empty label keys", func(t *testing.T) {
		config := `---
replicated_api_version: "2.7.0"
swarm:
  configs:
  - name: foo
    value: bar
    labels:
      "": bar
`
		var root RootConfig
		if err := yaml.Unmarshal([]byte(config), &root); err != nil {
			t.Fatal(err)
		}
		err := v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Swarm.Configs[0].Labels": "mapkeylengthnonzero",
		}); err != nil {
			t.Error(err)
		}
	})
}
