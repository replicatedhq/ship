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
	Config        []byte
	ExpectedValue map[string]interface{}
}

type TestSuppliedParams struct {
	Name          string
	Config        []libyaml.ConfigGroup
	Release       map[string]interface{}
	ExpectedValue bool
}

func TestHeadlessDaemon(t *testing.T) {
	tests := []TestHeadless{
		{
			Name:          "basic",
			Config:        []byte(`{"spam": "eggs"}`),
			ExpectedValue: map[string]interface{}{"spam": "eggs"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
			err := fakeFS.WriteFile(".ship/state.json", test.Config, 0666)
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
			Config:        []libyaml.ConfigGroup{},
			ExpectedValue: false,
			Name:          "empty test",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, not required",
		},
		{
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
			ExpectedValue: true,
			Name:          "one group one item, required, no value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, required, value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, not required, hidden, no value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, required, not hidden, no value",
		},
		{
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
			ExpectedValue: true,
			Name:          "one group one item, required, not hidden, no value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, required, not hidden, value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, not required, not hidden, no value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group one item, required, hidden, value",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group two items",
		},
		{
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
			ExpectedValue: true,
			Name:          "one group two items, required",
		},
		{
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
			ExpectedValue: true,
			Name:          "one group two items, required",
		},
		{
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
			ExpectedValue: false,
			Name:          "one group two items, required",
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
			req.Equal(err != nil, test.ExpectedValue)
		})
	}
}
