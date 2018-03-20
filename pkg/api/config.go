package api

import "github.com/replicatedhq/libyaml"

type Config struct {
	V1 []libyaml.ConfigGroup `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}
