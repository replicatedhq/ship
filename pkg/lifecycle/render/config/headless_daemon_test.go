package config

import (
	"context"
	"testing"

	"time"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type TestHeadless struct {
	Name          string
	State         []byte
	Release       *api.Release
	ExpectedValue map[string]interface{}
	ExpectedError bool
}

func TestHeadlessDaemon(t *testing.T) {
	tests := []TestHeadless{
		{
			Name:  "empty",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{},
					},
				},
			},
			ExpectedValue: map[string]interface{}{},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, not required, no value",
			State: []byte(`{"alpha": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: false,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": ""},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, required, no value",
			State: []byte(`{"alpha": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{},
			ExpectedError: true,
		},
		{
			Name:  "one group one item, required, value, hidden",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: true,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100"},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, not required, no value, hidden",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: false,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100"},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, required, no value, hidden",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": ""},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, required, no value, not hidden",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": ""},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, required, value, not hidden",
			State: []byte(`{"alpha": "100"}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100"},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, not required, no value, not hidden",
			State: []byte(`{"alpha": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: false,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": ""},
			ExpectedError: false,
		},
		{
			Name:  "one group one item, required, value, hidden",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: true,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100"},
			ExpectedError: false,
		},
		{
			Name:  "one group two items, neither required, neither present",
			State: []byte(`{"alpha": "", "beta": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: false,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    "",
									Default:  "",
									Required: false,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "", "beta": ""},
			ExpectedError: false,
		},
		{
			Name:  "one group two items, both required, neither present",
			State: []byte(`{"alpha": "", "beta": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{},
			ExpectedError: true,
		},
		{
			Name:  "one group two items, both required, one present",
			State: []byte(`{"alpha": "", "beta": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    "",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{},
			ExpectedError: true,
		},
		{
			Name:  "one group two items, both required, both present",
			State: []byte(`{"alpha": "", "beta": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    "200",
									Default:  "",
									Required: true,
									Hidden:   false,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100", "beta": "200"},
			ExpectedError: false,
		},
		{
			Name:  "beta value resolves to alpha value",
			State: []byte(`{"alpha": ""}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{{
							Name: "testing",
							Items: []*libyaml.ConfigItem{
								{
									Name:     "alpha",
									Value:    "100",
									Default:  "",
									Required: false,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    `{{repl ConfigOption "alpha" }}`,
									Default:  "",
									Required: false,
									Hidden:   true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: map[string]interface{}{"alpha": "100", "beta": "100"},
			ExpectedError: false,
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

			resolveContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err = daemon.HeadlessResolve(resolveContext, test.Release)
			if test.ExpectedError {
				req.Error(err)
			} else {
				updatedState, err := fakeFS.ReadFile(".ship/state.json")
				req.NoError(err)

				req.Equal(updatedState, test.ExpectedValue)
			}
		})
	}
}
