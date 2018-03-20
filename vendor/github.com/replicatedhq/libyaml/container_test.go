package libyaml_test

import (
	"testing"

	"github.com/replicatedhq/libyaml"

	yaml "gopkg.in/yaml.v2"
)

func TestContainerUnmarshalYAML(t *testing.T) {
	s := `source: public
image_name: test
display_name: Test Container
version: ""
privileged: false
hostname: ""
cmd: ""
cluster: false
publish_events: []
config_files: []
customer_files: []
env_vars: []
logs:
  max_size: "100k"
  max_files: "5"
volumes: []
support_files: []
support_commands: []`

	var c libyaml.Container
	if err := yaml.Unmarshal([]byte(s), &c); err != nil {
		t.Fatal(err)
	}

	if c.Source != "public" {
		t.Errorf("expecting \"Container.Source\" == \"public\", got \"%s\"", c.Source)
	}
	if c.ImageName != "test" {
		t.Errorf("expecting \"Container.ImageName\" == \"test\", got \"%s\"", c.ImageName)
	}
	if c.DisplayName != "Test Container" {
		t.Errorf("expecting \"Container.DisplayName\" == \"Test Container\", got \"%s\"", c.ImageName)
	}
	if c.Cluster != "false" {
		t.Error("expecting \"Container.Cluster\" to be \"false\"")
	}
	if c.ClusterInstanceCount.Initial != "" {
		t.Errorf("expecting \"Container.ClusterInstanceCount.Initial\" == \"\", got \"%s\"", c.ClusterInstanceCount.Initial)
	}
	if c.ClusterInstanceCount.ThresholdHealthy != "" {
		t.Errorf("expecting \"Container.ClusterInstanceCount.ThresholdHealthy\" == \"\", got \"%s\"", c.ClusterInstanceCount.ThresholdHealthy)
	}
	if c.LogOptions.MaxFiles != "5" {
		t.Errorf("expecting \"Container.MaxFiles\" == \"5\", got \"%s\"", c.LogOptions.MaxFiles)
	}
	if c.LogOptions.MaxSize != "100k" {
		t.Errorf("expecting \"Container.MaxSize\" == \"100k\", got \"%s\"", c.LogOptions.MaxSize)
	}
	if c.Entrypoint != nil {
		t.Errorf("expecting \"Container.Entrypoint\" == \"nil\", got \"%v\"", c.Entrypoint)
	}
}

func TestContainerUnmarshalYAMLCluster(t *testing.T) {
	s := `source: public
image_name: test
version: ""
display_name: Test Container
privileged: false
hostname: ""
cmd: ""
entrypoint: []
cluster: true
publish_events: []
config_files: []
customer_files: []
env_vars: []
logs:
  max_size: ""
  max_files: ""
volumes: []
support_files: []
support_commands: []`

	var c libyaml.Container
	if err := yaml.Unmarshal([]byte(s), &c); err != nil {
		t.Fatal(err)
	}

	if c.Source != "public" {
		t.Errorf("expecting \"Container.Source\" == \"public\", got \"%s\"", c.Source)
	}
	if c.ImageName != "test" {
		t.Errorf("expecting \"Container.ImageName\" == \"test\", got \"%s\"", c.ImageName)
	}
	if c.DisplayName != "Test Container" {
		t.Errorf("expecting \"Container.DisplayName\" == \"Test Container\", got \"%s\"", c.ImageName)
	}
	if c.Cluster != "true" {
		t.Error("expecting \"Container.Cluster\" to be \"true\"")
	}
	if c.ClusterInstanceCount.Initial != "1" {
		t.Errorf("expecting \"Container.ClusterInstanceCount.Initial\" == 1, got \"%s\"", c.ClusterInstanceCount.Initial)
	}
	if c.ClusterInstanceCount.ThresholdHealthy != "" {
		t.Errorf("expecting \"Container.ClusterInstanceCount.ThresholdHealthy\" == \"\", got \"%s\"", c.ClusterInstanceCount.ThresholdHealthy)
	}
	if c.Entrypoint == nil || len(*c.Entrypoint) != 0 {
		t.Errorf("expecting \"Container.Entrypoint\" to be empty, got \"%v\"", c.Entrypoint)
	}
}

func TestContainerMarshalYAML(t *testing.T) {
	s := `source: public
image_name: test
version: ""
display_name: ""
name: ""
privileged: false
network_mode: ""
cpu_shares: ""
memory_limit: ""
memory_swap_limit: ""
ulimits: []
allocate_tty: ""
security_cap_add: []
security_options: []
hostname: ""
cmd: ""
entrypoint: null
ephemeral: false
suppress_restart: []
cluster: false
restart: null
publish_events: []
config_files: []
customer_files: []
env_vars: []
logs:
  max_size: 100k
  max_files: "5"
volumes: []
volumes_from: []
extra_hosts: []
support_files: []
support_commands: []
content_trust:
  public_key_fingerprint: ""
when: ""
dynamic: ""
pid_mode: ""
shm_size: 0
`

	logReqs := libyaml.LogOptions{
		MaxSize:  "100k",
		MaxFiles: "5",
	}
	c := libyaml.Container{
		Source:     "public",
		ImageName:  "test",
		Cluster:    "false",
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

func TestContainerMarshalYAMLCluster(t *testing.T) {
	s := `source: public
image_name: test
version: ""
display_name: ""
name: ""
privileged: false
network_mode: ""
cpu_shares: ""
memory_limit: ""
memory_swap_limit: ""
allocate_tty: ""
security_cap_add: []
security_options: []
hostname: ""
cmd: ""
entrypoint: null
ephemeral: false
suppress_restart: []
cluster: true
restart: null
cluster_instance_count:
  initial: 1
publish_events: []
config_files: []
customer_files: []
env_vars: []
logs:
  max_size: ""
  max_files: ""
volumes: []
volumes_from: []
extra_hosts: []
support_files: []
support_commands: []
content_trust:
  public_key_fingerprint: ""
when: ""
dynamic: ""
pid_mode: ""
shm_size: 0
`

	c := libyaml.Container{
		Source:    "public",
		ImageName: "test",
		Cluster:   "true",
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != s {
		t.Errorf("unexpected marshalled YAML,\nexpecting\n%s\ngot\n%s", s, string(b))
	}
}
