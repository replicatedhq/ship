package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mitchellh/cli"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/headless"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/state"
	templates "github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type daemonResolverTestCase struct {
	name         string
	release      *api.Release
	inputContext map[string]interface{}
	posts        []func(t *testing.T) //nolint:structcheck
	expect       func(t *testing.T, config map[string]interface{}, err error)
}

func TestDaemonResolver(t *testing.T) {
	tests := []daemonResolverTestCase{
		{
			name: "test_resolve_noitems",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{},
					},
				},
			},
			inputContext: map[string]interface{}{
				"foo": "bar",
			},
			expect: func(t *testing.T, config map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)
				actual, ok := config["foo"]
				req.True(ok, "Expected to find key %s in resolved config", "foo")
				req.Equal("bar", actual)
			},
		},
		{
			name: "test_resolve_timeout",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name: "k8s_namespace",
										Type: "text",
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				require.New(t).Error(e)
			},
		},
		{
			name: "test_single_item",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name: "k8s_namespace",
										Type: "text",
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{},
			posts: []func(t *testing.T){
				func(t *testing.T) {
					//http.Post("")

				},
			},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				// todo this should not fail
				require.New(t).Error(e)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := viper.New()

			viper.Set("api-port", 0)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			log := &logger.TestLogger{T: t}

			daemon := &daemon.ShipDaemon{
				Logger:       log,
				WebUIFactory: daemon.WebUIFactoryFactory(log),
				Viper:        v,
				V1Routes: &daemon.V1Routes{
					Logger: log,
					Fs:     fs,
					Viper:  v,

					UI:             cli.NewMockUi(),
					OpenWebConsole: func(ui cli.Ui, s string, b bool) error { return nil },
				},
				NavcycleRoutes: &daemon.NavcycleRoutes{
					Kustomizer: &kustomize.Kustomizer{},
					Shutdown:   make(chan interface{}),
				},
			}

			daemonCtx, daemonCancelFunc := context.WithCancel(context.Background())
			daemonCloseChan := make(chan struct{})

			require.NoError(t, log.Log("starting daemon"))
			go func(closeChan chan struct{}) {
				_ = daemon.Serve(daemonCtx, test.release)
				closeChan <- struct{}{}
			}(daemonCloseChan)

			resolver := &DaemonResolver{log, daemon}

			resolveContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			config, err := resolver.ResolveConfig(resolveContext, test.release, test.inputContext)

			daemonCancelFunc()
			cancel()

			<-daemonCloseChan

			test.expect(t, config, err)

			//req.NoError(err)
			//
			//for key, expected := range test.expect {
			//	actual, ok := config[key]
			//	req.True(ok, "Expected to find key %s in resolved config", key)
			//	req.Equal(expected, actual)
			//}
		})
	}
}

func TestHeadlessResolver(t *testing.T) {
	tests := []daemonResolverTestCase{
		{
			name: "test_resolve_noitems",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{},
					},
				},
			},
			inputContext: map[string]interface{}{
				"foo": "bar",
			},
			expect: func(t *testing.T, config map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)
				actual, ok := config["foo"]
				req.True(ok, "Expected to find key %s in resolved config", "foo")
				req.Equal("bar", actual)
			},
		},
		{
			name: "test_config_item",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "out",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "foo"}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "foo",
										Type:     "text",
										ReadOnly: false,
										Value:    ``,
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{
				"foo": "bar",
			},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)

				expectContext := map[string]interface{}{
					"foo": "bar",
					"out": "bar",
				}

				req.Equal(expectContext, i)
			},
		},
		{
			name: "test_random_chain",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_1",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl RandomString 32}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_dependent",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "random_1"}}`,
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)

				random1, exists := i["random_1"]
				req.True(exists, "'random_1' should exist")

				randomDependent, exists := i["random_dependent"]
				req.True(exists, "'random_dependent' should exist")

				req.Equal(randomDependent, random1)
			},
		},
		{
			name: "test_deep_random_chain",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_1",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl RandomString 32}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_dependent",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "random_1"}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_dependent_child",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "random_dependent"}}+{{repl ConfigOption "random_dependent"}}`,
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)

				random1, exists := i["random_1"]
				req.True(exists, "'random_1' should exist")

				randomDependent, exists := i["random_dependent"]
				req.True(exists, "'random_dependent' should exist")

				randomDependentChild, exists := i["random_dependent_child"]
				req.True(exists, "'random_dependent_child' should exist")

				req.Equal(randomDependentChild, fmt.Sprintf("%s+%s", randomDependent, randomDependent), "constructed child should match")
				req.Equal(randomDependent, random1, "raw child should match")
			},
		},
		{
			name: "ensure_randomstrings_differ",
			release: &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: []api.Step{
							{
								Render: &api.Render{},
							},
						},
					},
					Config: api.Config{
						V1: []libyaml.ConfigGroup{
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_1",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl RandomString 32}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_dependent",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "random_1"}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_2",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl RandomString 32}}`,
									},
								},
							},
							{
								Items: []*libyaml.ConfigItem{
									{
										Name:     "random_dependent_2",
										Type:     "text",
										ReadOnly: true,
										Value:    `{{repl ConfigOption "random_2"}}`,
									},
								},
							},
						},
					},
				},
			},
			inputContext: map[string]interface{}{},
			expect: func(t *testing.T, i map[string]interface{}, e error) {
				req := require.New(t)
				req.NoError(e)

				random1, exists := i["random_1"]
				req.True(exists, "'random_1' should exist")

				randomDependent, exists := i["random_dependent"]
				req.True(exists, "'random_dependent' should exist")

				req.Equal(randomDependent, random1)

				random2, exists := i["random_2"]
				req.True(exists, "'random_2' should exist")

				randomDependent2, exists := i["random_dependent_2"]
				req.True(exists, "'random_dependent_2' should exist")

				req.Equal(randomDependent2, random2)

				req.NotEqual(random1, random2)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			v := viper.New()

			viper.Set("api-port", 0)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			log := &logger.TestLogger{T: t}

			manager, err := state.NewDisposableManager(log, fs, v)
			req.NoError(err)

			builderBuilder := templates.NewBuilderBuilder(log, v, manager)

			renderer := resolve.NewRenderer(log, v, builderBuilder)

			headlessDaemon := headless.HeadlessDaemon{
				StateManager:      manager,
				Logger:            log,
				UI:                ui.FromViper(v),
				ConfigRenderer:    renderer,
				FS:                fs,
				ResolvedConfig:    test.inputContext,
				YesApplyTerraform: false,
			}

			resolver := &DaemonResolver{log, &headlessDaemon}

			resolveContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)

			config, err := resolver.ResolveConfig(resolveContext, test.release, test.inputContext)

			test.expect(t, config, err)
			cancel()
		})
	}
}
