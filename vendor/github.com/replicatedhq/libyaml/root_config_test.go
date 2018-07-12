package libyaml_test

import . "github.com/replicatedhq/libyaml"

func newRootConfig() *RootConfig {
	return &RootConfig{
		APIVersion: "2.4.0",
	}
}
