package config

import (
	"testing"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type TestHeadless struct {
	Name          string
	State         []byte
	ExpectedValue map[string]interface{}
}

type TestSuppliedParams struct {
	Name          string
	Config        []libyaml.ConfigGroup
	ExpectedError bool
}

type TestRenderChainedConfigOpts struct {
	Name          string
	State         []byte
	Config        []libyaml.ConfigGroup
	ExpectedValue map[string]interface{}
}

func TestHeadlessDaemon(t *testing.T) {
	tests := []TestHeadless{
		{
			Name:          "empty",
			State:         []byte(`{}`),
			ExpectedValue: map[string]interface{}{},
		},
		{
			Name:          "basic",
			State:         []byte(`{"spam": "eggs"}`),
			ExpectedValue: map[string]interface{}{"spam": "eggs"},
		},
		{
			Name:          "multiple",
			State:         []byte(`{"spam": "eggs", "ford": "bernard"}`),
			ExpectedValue: map[string]interface{}{"spam": "eggs", "ford": "bernard"},
		},
		{
			Name:          "some empty fields",
			State:         []byte(`{"spam": "", "ford": "bernard"}`),
			ExpectedValue: map[string]interface{}{"spam": "", "ford": "bernard"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
			err := fakeFS.WriteFile(".ship/state.json", test.State, 0666)
			req.NoError(err)

			testLogger := &logger.TestLogger{T: t}
			daemon := &HeadlessDaemon{
				StateManager: &state.StateManager{
					Logger: testLogger,
					FS:     fakeFS,
				},
				Logger: testLogger,
			}

			cfg := daemon.GetCurrentConfig()
			req.Equal(cfg, test.ExpectedValue)
		})
	}
}

func TestValidateSuppliedParams(t *testing.T) {
	tests := []TestSuppliedParams{
		{
			Name:          "empty test",
			Config:        []libyaml.ConfigGroup{},
			ExpectedError: false,
		},
		{
			Name: "one group one item, not required, no value",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: false,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, required, no value",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: true,
		},
		{
			Name: "one group one item, required, value",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "abc",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, not required, no value, hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: false,
							Value:    "",
							Default:  "",
							Hidden:   true,
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, required, no value, hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "",
							Default:  "",
							Hidden:   true,
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, required, no value, not hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "",
							Default:  "",
							Hidden:   false,
						},
					},
				},
			},
			ExpectedError: true,
		},
		{
			Name: "one group one item, required, value, not hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "abc",
							Default:  "",
							Hidden:   false,
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, not required, no value, not hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: false,
							Value:    "",
							Default:  "",
							Hidden:   false,
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group one item, required, value, hidden",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Required: true,
							Value:    "abc",
							Default:  "",
							Hidden:   true,
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group two items, neither required, neither present",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Required: false,
							Value:    "",
							Default:  "",
						},
						{
							Name:     "beta",
							Required: false,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: false,
		},
		{
			Name: "one group two items, both required, neither present",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Required: true,
							Value:    "",
							Default:  "",
						},
						{
							Name:     "beta",
							Required: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: true,
		},
		{
			Name: "one group two items, both required, one present",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Required: true,
							Value:    "abc",
							Default:  "",
						},
						{
							Name:     "beta",
							Required: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: true,
		},
		{
			Name: "one group two items, both required, both present",
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Required: true,
							Value:    "abc",
							Default:  "",
						},
						{
							Name:     "beta",
							Required: true,
							Value:    "xyz",
							Default:  "",
						},
					},
				},
			},
			ExpectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}

			testLogger := &logger.TestLogger{T: t}
			daemon := &HeadlessDaemon{
				StateManager: &state.StateManager{
					Logger: testLogger,
					FS:     fakeFS,
				},
				Logger: testLogger,
			}

			err := daemon.ValidateSuppliedParams(test.Config)
			if test.ExpectedError {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}

func TestChainedConfig(t *testing.T) {
	tests := []TestRenderChainedConfigOpts{
		{
			Name:          "empty",
			State:         []byte(`{}`),
			Config:        []libyaml.ConfigGroup{},
			ExpectedValue: map[string]interface{}{},
		},
		{
			Name:  "beta value resolves to alpha value",
			State: []byte(`{"alpha": "100"}`),
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Required: false,
							Value:    "100",
							Default:  "",
							Hidden:   false,
						},
						{
							Name:     "beta",
							Required: false,
							Value:    `{{repl ConfigOption "alpha" }}`,
							Default:  "",
							Hidden:   true,
						},
					},
				}},
			ExpectedValue: map[string]interface{}{
				"alpha": "100",
				"beta":  "100",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
			err := fakeFS.WriteFile(".ship/state.json", test.State, 0666)
			req.NoError(err)

			// testLogger := &logger.TestLogger{T: t}
			// daemon := &HeadlessDaemon{
			// 	StateManager: &state.StateManager{
			// 		Logger: testLogger,
			// 		FS:     fakeFS,
			// 	},
			// 	Logger: testLogger,
			// }

			// cfg := daemon.GetCurrentConfig()
			// req.Equal(cfg, test.ExpectedValue)
		})
	}
}
