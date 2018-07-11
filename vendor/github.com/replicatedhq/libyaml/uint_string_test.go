package libyaml_test

import (
	"testing"

	"github.com/replicatedhq/libyaml"
	"gopkg.in/yaml.v2"
)

func TestUintStringMarshal(t *testing.T) {
	type S struct {
		Uint libyaml.UintString
	}
	s := S{Uint: libyaml.UintString("1")}
	b, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "uint: 1\n" {
		t.Errorf(`expecting "uint: 1\n", got %q`, b)
	}

	s = S{Uint: libyaml.UintString("")}
	b, err = yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "uint: 0\n" {
		t.Errorf(`expecting "uint: 0\n", got %q`, b)
	}

	s = S{Uint: libyaml.UintString("{{repl blah }}")}
	b, err = yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "uint: '{{repl blah }}'\n" {
		t.Errorf(`expecting "uint: '{{repl blah }}'\n", got %q`, b)
	}
}
