package libyaml_test

import (
	"strings"
	"testing"

	"github.com/replicatedhq/libyaml"
	validator "gopkg.in/go-playground/validator.v8"
	"gopkg.in/yaml.v2"
)

func TestBoolStringMarshal(t *testing.T) {
	type S struct {
		Bool libyaml.BoolString `yaml:"is_true"`
	}
	s := S{Bool: libyaml.BoolString("true")}
	b, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "is_true: true\n" {
		t.Errorf(`expecting "is_true: true\n", got %q`, b)
	}

	s = S{Bool: libyaml.BoolString("")}
	b, err = yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "is_true: false\n" {
		t.Errorf(`expecting "is_true: true\n", got %q`, b)
	}

	s = S{Bool: libyaml.BoolString("{{repl blah }}")}
	b, err = yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "is_true: '{{repl blah }}'\n" {
		t.Errorf(`expecting "is_true: '{{repl blah }}'\n", got %q`, b)
	}
}

func TestBoolStringValidate(t *testing.T) {
	v := validator.New(
		&validator.Config{TagName: "validate"},
	)
	err := v.RegisterValidation("bool", libyaml.IsBoolValidation)
	if err != nil {
		t.Fatal(err)
	}
	type S struct {
		Bool libyaml.BoolString `yaml:"is_true" validate:"bool"`
	}
	s := S{Bool: libyaml.BoolString("true")}
	err = v.Struct(s)
	if err != nil {
		t.Fatal(err)
	}

	s = S{Bool: libyaml.BoolString("0")}
	err = v.Struct(s)
	if err != nil {
		t.Fatal(err)
	}

	s = S{Bool: libyaml.BoolString("{{repl blah }}")}
	err = v.Struct(s)
	if err != nil {
		t.Fatal(err)
	}

	s = S{Bool: libyaml.BoolString("blah")}
	err = v.Struct(s)
	if err == nil {
		t.Fatal(err)
	}
	if !strings.Contains(err.Error(), "failed on the 'bool' tag") {
		t.Errorf("unexpected error, %v", err)
	}

	s = S{Bool: libyaml.BoolString("")}
	err = v.Struct(s)
	if !strings.Contains(err.Error(), "failed on the 'bool' tag") {
		t.Errorf("unexpected error, %v", err)
	}
}
