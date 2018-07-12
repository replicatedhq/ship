package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
)

func TestCustomRequirementIDUnique(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	func(t *testing.T) {
		root := newRootConfig()
		root.CustomRequirements = []CustomRequirement{
			newCustomRequirement("1"),
			newCustomRequirement("2"),
		}
		err := v.Struct(root)
		if err != nil {
			t.Error(err)
		}
	}(t)

	func(t *testing.T) {
		root := newRootConfig()
		root.CustomRequirements = []CustomRequirement{
			newCustomRequirement("1"),
			newCustomRequirement("2"),
			newCustomRequirement("1"),
		}
		err := v.Struct(root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.CustomRequirements[0].ID": "customrequirementidunique",
			"RootConfig.CustomRequirements[2].ID": "customrequirementidunique",
		}); err != nil {
			t.Error(err)
		}
	}(t)
}
