package config

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"os"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedcom/ship/pkg/api"
	_ "github.com/replicatedcom/ship/pkg/lifecycle/render/config/test-cases"
	"github.com/replicatedcom/ship/pkg/test-mocks/ui"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type cliTestcase struct {
	Name         string
	Config       []libyaml.ConfigGroup
	OSEnv        map[string]string      `yaml:"osenv"`
	ViperConfig  map[string]interface{} `yaml:"viper_config"`
	Responses    []cliExpectUIAsk       `yaml:"responses"`
	Expect       map[string]string
	ExpectUIInfo []string `yaml:"expect_ui_info"`
	ExpectUIWarn []string `yaml:"expect_ui_warn"`
}

type cliExpectUIAsk struct {
	Question string
	Answer   string
}

func TestCLIResolver(t *testing.T) {
	ctx := context.Background()

	logger := log.NewLogfmtLogger(os.Stderr)
	resolver := &CLIResolver{
		Logger: logger,
	}

	tests := loadCLITestCases(t, filepath.Join("test-cases", "config-test-cli.yml"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockUI := ui.NewMockUi(mc)

			release := &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: test.Config,
					},
				},
			}
			resolver.UI = mockUI

			resolver.Viper = viper.New()
			resolver.Viper.AutomaticEnv()

			func() {
				defer mc.Finish()

				fmt.Println(test.OSEnv)

				for key, value := range test.OSEnv {
					old := os.Getenv(key)
					err := os.Setenv(key, value)
					req.NoError(err)
					defer func(key, old, value string) {
						os.Setenv(key, old)
					}(key, old, value)
				}
				for _, expected := range test.ExpectUIInfo {
					mockUI.EXPECT().Info(expected)
				}

				for _, expected := range test.ExpectUIWarn {
					mockUI.EXPECT().Warn(expected)
				}

				for _, expect := range test.Responses {
					mockUI.EXPECT().Ask(expect.Question).Return(expect.Answer, nil)
				}

				resolvedConfig, err := resolver.ResolveConfig(ctx, release, nil)
				req.NoError(err)

				for key, expected := range test.Expect {
					actual, ok := resolvedConfig[key]
					req.True(ok, "Expected to find key %s in resolved config", key)
					req.Equal(expected, actual)
				}
			}()
		})
	}
}

func loadCLITestCases(t *testing.T, path string) []cliTestcase {
	tests := make([]cliTestcase, 1)
	contents, err := ioutil.ReadFile(path)
	assert.NoError(t, err)
	err = yaml.UnmarshalStrict(contents, &tests)
	assert.NoError(t, err)
	return tests
}
