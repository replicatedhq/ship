package libyaml_test

import (
	"testing"

	. "github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	yaml "gopkg.in/yaml.v2"
)

func TestAirgapImages(t *testing.T) {
	v := validator.New(&validator.Config{TagName: "validate"})
	err := RegisterValidations(v)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
images:
- name: redis
  version: 3.2
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

	t.Run("no name", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
images:
- source: public
  version: 3.2
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Images[0].Name": "required",
		}); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid fingerprint", func(t *testing.T) {
		config := `---
replicated_api_version: "2.8.0"
images:
- source: public
  name: redis
  version: 3.2
  content_trust:
    public_key_fingerprint: blah
`
		var root RootConfig
		err := yaml.Unmarshal([]byte(config), &root)
		if err != nil {
			t.Fatal(err)
		}
		err = v.Struct(&root)
		if err := AssertValidationErrors(t, err, map[string]string{
			"RootConfig.Images[0].ContentTrust.PublicKeyFingerprint": "fingerprint",
		}); err != nil {
			t.Error(err)
		}
	})
}
