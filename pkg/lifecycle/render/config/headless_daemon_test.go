package config

import (
	"context"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type TestHeadless struct {
	Name          string
	State         []byte
	Release       *api.Release
	ExpectedValue []byte
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
			ExpectedValue: []byte(`{"v1":{"config":{}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":""}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100"}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100"}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":""}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":""}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100"}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":""}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100"}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"","beta":""}}}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{}}}`),
			ExpectedError: true,
		},
		{
			Name:  "one group two items, both required, one present",
			State: []byte(`{"alpha":"", "beta": ""}`),
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"","beta":""}}}`),
			ExpectedError: true,
		},
		{
			Name:  "one group two items, both required, both present",
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
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100","beta":"200"}}}`),
			ExpectedError: false,
		},
		{
			Name:  "beta value resolves to alpha value",
			State: []byte(`{"alpha": "101"}`),
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
									ReadOnly: true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"101","beta":"101"}}}`),
			ExpectedError: false,
		},
		{
			Name:  "beta value resolves to alpha value when wrong beta value is presented",
			State: []byte(`{"alpha": "101", "beta":"abc"}`),
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
									ReadOnly: true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"101","beta":"101"}}}`),
			ExpectedError: false,
		},
		{
			Name:  "charlie value resolves to beta value resolves to alpha value",
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
									Required: false,
									Hidden:   false,
								},
								{
									Name:     "beta",
									Value:    `{{repl ConfigOption "alpha" }}`,
									Default:  "",
									Required: false,
									ReadOnly: true,
								},
								{
									Name:     "charlie",
									Value:    `{{repl ConfigOption "beta" }}`,
									Default:  "",
									Required: false,
									ReadOnly: true,
								},
							},
						}},
					},
				},
			},
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100","beta":"100","charlie":"100"}}}`),
			ExpectedError: false,
		},
		{
			Name:  "multiple groups with multiple items",
			State: []byte(`{}`),
			Release: &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Name: "testing",
								Items: []*libyaml.ConfigItem{
									{
										Name:     "cluster info",
										Value:    "",
										Default:  "",
										Required: true,
										Hidden:   false,
									},
									{
										Name:     "worker replicas",
										Value:    "",
										Default:  "",
										Required: true,
										Hidden:   false,
									},
								},
							},
							{
								Name: "testing",
								Items: []*libyaml.ConfigItem{
									{
										Name:     "semver",
										Value:    "",
										Default:  "",
										Required: true,
										Hidden:   false,
									},
								},
							},
							{
								Name: "testing",
								Items: []*libyaml.ConfigItem{
									{
										Name:     "alpha",
										Value:    "hello world",
										Default:  "",
										Required: false,
										Hidden:   false,
									},
									{
										Name:     "beta",
										Value:    `{{repl ConfigOption "alpha" }}`,
										Default:  "",
										Required: false,
										ReadOnly: true,
										Hidden:   true,
									},
									{
										Name:     "charlie",
										Value:    `{{repl ConfigOption "beta" }}`,
										Default:  "",
										Required: false,
										ReadOnly: true,
										Hidden:   false,
									},
								},
							},
						},
					},
				},
			},
			ExpectedValue: []byte(`{"v1":{"config":{"alpha":"100","beta":"100","charlie":"100"}}}`),
			ExpectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}

			err := fakeFS.WriteFile(".ship/state.json", test.State, 0666)
			req.NoError(err)

			v := viper.New()
			testLogger := &logger.TestLogger{T: t}

			builder := &templates.BuilderBuilder{
				Logger: testLogger,
				Viper:  v,
			}

			manager := &state.Manager{
				Logger: testLogger,
				FS:     fakeFS,
				V:      viper.New(),
			}

			resolver := &APIConfigRenderer{
				Logger:         testLogger,
				Viper:          v,
				BuilderBuilder: builder,
			}

			daemon := &HeadlessDaemon{
				StateManager:   manager,
				Logger:         testLogger,
				ConfigRenderer: resolver,
				UI:             cli.NewMockUi(),
			}

			ctx := context.Background()

			err = daemon.HeadlessResolve(ctx, test.Release)
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
