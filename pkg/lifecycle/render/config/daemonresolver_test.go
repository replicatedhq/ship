package config

import (
	"context"
	"testing"
	"time"

	"github.com/mitchellh/cli"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type daemonResolverTestCase struct {
	name         string
	release      *api.Release
	inputContext map[string]interface{}
	posts        []func(t *testing.T)
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
				Logger: log,
				Fs:     fs,
				Viper:  v,

				UI:             cli.NewMockUi(),
				WebUIFactory:   daemon.WebUIFactoryFactory(log),
				OpenWebConsole: func(ui cli.Ui, s string) error { return nil },
			}

			daemonCtx, daemonCancelFunc := context.WithCancel(context.Background())
			defer daemonCancelFunc()

			log.Log("starting daemon")
			go func() {
				daemon.Serve(daemonCtx, test.release)
			}()

			resolver := &DaemonResolver{log, daemon}

			resolveContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			config, err := resolver.ResolveConfig(resolveContext, test.release, test.inputContext)

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
