package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	yaml "gopkg.in/yaml.v2"
)

func TestCustomRequirement(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	// test success
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
  command:
    id: cmd-id
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
	}(t)

	// test message and message pointer required
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  details:
    default_message:
  results:
  - status: status
    message: message
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].Message.DefaultMessage": "required",
			"RootConfig.CustomRequirements[0].Details.DefaultMessage": "required",
		}); err != nil {
			t.Error(err)
		}
	}(t)

	// test id unique
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
  command:
    id: cmd-id
- id: req-id
  message: message
  results:
  - status: status
    message: message
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].ID": "customrequirementidunique",
			"RootConfig.CustomRequirements[1].ID": "customrequirementidunique",
		}); err != nil {
			t.Error(err)
		}
	}(t)
}

func TestCustomRequirementResults(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	// test success
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
    condition:
      error: true
      status_code: 1
      bool_expr: true
  command:
    id: cmd-id
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
	}(t)

	// test required
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].Results": "required",
		}); err != nil {
			t.Error(err)
		}
	}(t)

	// test status code
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
    condition:
      status_code: 1
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		actual := *(root.CustomRequirements[0].Results[0].Condition.StatusCode)
		if actual != 1 {
			t.Errorf("expecting \"StatusCode\" == 1, got %d", actual)
		}

		config = `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
    condition:
      status_code: 0
  command:
    id: cmd-id
`
		err = yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		actual = *(root.CustomRequirements[0].Results[0].Condition.StatusCode)
		if actual != 0 {
			t.Errorf("expecting \"StatusCode\" == 0, got %d", actual)
		}

		config = `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
    condition:
      error: true
  command:
    id: cmd-id
`
		err = yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		if root.CustomRequirements[0].Results[0].Condition.StatusCode != nil {
			actual = *(root.CustomRequirements[0].Results[0].Condition.StatusCode)
			t.Errorf("expecting \"StatusCode\" nil, got %d", actual)
		}
	}(t)

	// test min
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results: []
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].Results": "min",
		}); err != nil {
			t.Error(err)
		}
	}(t)

	// test dive required
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status:
    message:
  command:
    id: cmd-id
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].Results[0].Status":                 "required",
			"RootConfig.CustomRequirements[0].Results[0].Message.DefaultMessage": "required",
		}); err != nil {
			t.Error(err)
		}
	}(t)
}

func TestCustomRequirementCommand(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	// test id required
	func(t *testing.T) {
		config := `---
replicated_api_version: "1.3.2"
custom_requirements:
- id: req-id
  message: message
  results:
  - status: status
    message: message
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].Command.ID": "required",
		}); err != nil {
			t.Error(err)
		}
	}(t)
}

func newCustomRequirement(id string) CustomRequirement {
	return CustomRequirement{
		ID: id,
		Message: Message{
			DefaultMessage: "message",
		},
		Results: []CustomResult{
			{
				Status: "status",
				Message: Message{
					DefaultMessage: "message",
				},
			},
		},
		Command: CustomCommand{
			ID: "id",
		},
	}
}
