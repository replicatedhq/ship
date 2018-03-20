package libyaml_test

import (
	"reflect"
	"testing"

	. "github.com/replicatedhq/libyaml"
	yaml "gopkg.in/yaml.v2"
)

func TestMessageUnmarshalYAML(t *testing.T) {
	var m *Message

	m = &Message{}
	data := `
id: message-id
default_message: message default
args:
  a: A
  one: 1
`
	err := yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		t.Fatal(err)
	}
	expected := Message{
		ID:             "message-id",
		DefaultMessage: "message default",
		Args:           map[string]interface{}{"a": "A", "one": uint64(1)},
	}
	if reflect.DeepEqual(expected, *m) {
		t.Errorf(`Expecting Message %q, got %q`, expected, *m)
	}

	m = &Message{}
	data = `message default`
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		t.Fatal(err)
	}
	expected = Message{
		DefaultMessage: "message default",
		Args:           map[string]interface{}{},
	}
	if reflect.DeepEqual(expected, *m) {
		t.Errorf(`Expecting Message %q, got %q`, expected, *m)
	}
}

func TestMessageUnmarshalJSON(t *testing.T) {
	var m *Message

	m = &Message{}
	data := `{"id": "message-id", "default_message": "message default", "args": {"a": "A", "one": 1}}`
	err := yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		t.Fatal(err)
	}
	expected := Message{
		ID:             "message-id",
		DefaultMessage: "message default",
		Args:           map[string]interface{}{"a": "A", "one": uint64(1)},
	}
	if reflect.DeepEqual(expected, *m) {
		t.Errorf(`Expecting Message %q, got %q`, expected, *m)
	}

	m = &Message{}
	data = `message default`
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		t.Fatal(err)
	}
	expected = Message{
		DefaultMessage: "message default",
		Args:           map[string]interface{}{},
	}
	if reflect.DeepEqual(expected, *m) {
		t.Errorf(`Expecting Message %q, got %q`, expected, *m)
	}
}
