package libyaml_test

import (
	"testing"

	"github.com/replicatedhq/libyaml"

	yaml "gopkg.in/yaml.v2"
)

func TestComponentUnmarshalYAML(t *testing.T) {
	s := `name: test
cluster: false
logs:
  max_size: 100k
  max_files: 5
containers: []`

	var c libyaml.Component
	if err := yaml.Unmarshal([]byte(s), &c); err != nil {
		t.Fatal(err)
	}

	if c.Name != "test" {
		t.Errorf("expecting \"Component.Name\" == \"test\", got \"%s\"", c.Name)
	}
	if c.Cluster != "false" {
		t.Error("expecting \"Component.Cluster\" to be \"false\"")
	}
	if c.ClusterHostCount.Min != "" {
		t.Errorf("expecting \"Component.ClusterHostCount.Min\" == \"\", got \"%s\"", c.ClusterHostCount.Min)
	}
	if c.ClusterHostCount.ThresholdHealthy != "" {
		t.Errorf("expecting \"Component.ClusterHostCount.ThresholdHealthy\" == \"\", got \"%s\"", c.ClusterHostCount.ThresholdHealthy)
	}
	if c.LogOptions.MaxFiles != "5" {
		t.Errorf("expecting \"Component.MaxFiles\" == \"5\", got \"%s\"", c.LogOptions.MaxFiles)
	}
	if c.LogOptions.MaxSize != "100k" {
		t.Errorf("expecting \"Component.MaxSize\" == \"100k\", got \"%s\"", c.LogOptions.MaxSize)
	}
}

func TestComponentUnmarshalYAMLCluster(t *testing.T) {
	s := `name: test
cluster: true
containers: []`

	var c libyaml.Component
	if err := yaml.Unmarshal([]byte(s), &c); err != nil {
		t.Fatal(err)
	}

	if c.Name != "test" {
		t.Errorf("expecting \"Component.Name\" == \"test\", got \"%s\"", c.Name)
	}
	if c.Cluster != "true" {
		t.Error("expecting \"Component.Cluster\" to be \"true\"")
	}
	if c.ClusterHostCount.Min != "" {
		t.Errorf("expecting \"Component.ClusterHostCount.Min\" == \"\", got \"%s\"", c.ClusterHostCount.Min)
	}
	if c.ClusterHostCount.ThresholdHealthy != "" {
		t.Errorf("expecting \"Component.ClusterHostCount.ThresholdHealthy\" == \"\", got \"%s\"", c.ClusterHostCount.ThresholdHealthy)
	}
}

func TestComponentMarshalYAML(t *testing.T) {
	s := `name: test
cluster: false
logs:
  max_size: ""
  max_files: ""
containers: []
`

	c := libyaml.Component{
		Name:    "test",
		Cluster: "false",
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != s {
		t.Errorf("unexpected marshalled YAML,\nexpecting\n%s\ngot\n%s", s, string(b))
	}
}

func TestComponentMarshalYAMLCluster(t *testing.T) {
	s := `name: test
cluster: true
logs:
  max_size: 100k
  max_files: "5"
containers: []
`

	logReqs := libyaml.LogOptions{
		MaxSize:  "100k",
		MaxFiles: "5",
	}
	c := libyaml.Component{
		Name:       "test",
		Cluster:    "true",
		LogOptions: logReqs,
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != s {
		t.Errorf("unexpected marshalled YAML,\nexpecting\n%s\ngot\n%s", s, string(b))
	}
}
