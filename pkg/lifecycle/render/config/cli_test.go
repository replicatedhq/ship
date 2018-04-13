package config

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedcom/ship/pkg/api"
	_ "github.com/replicatedcom/ship/pkg/lifecycle/render/config/test-fixtures"
	"github.com/replicatedcom/ship/pkg/test-fixtures/ui"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type testcase struct {
	Name         string
	Config       []libyaml.ConfigGroup
	ViperConfig  map[string]interface{} `yaml:"viper_config"`
	Responses    []expectUIAsk          `yaml:"responses"`
	Expect       map[string]string
	ExpectUIInfo []string `yaml:"expect_ui_info"`
	ExpectUIWarn []string `yaml:"expect_ui_warn"`
}

type expectUIAsk struct {
	Question string
	Answer   string
}

func TestCLIResolver(t *testing.T) {
	ctx := context.Background()

	resolver := &CLIResolver{
		Logger: log.NewNopLogger(),
	}

	tests := loadTestCases(t, filepath.Join("test-fixtures", "config-test.yml"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockUI := ui.NewMockUi(mc)

			resolver.Spec = &api.Spec{
				Config: api.Config{
					V1: test.Config,
				},
			}
			resolver.UI = mockUI
			resolver.Viper = viper.New()

			func() {
				defer mc.Finish()

				for _, expected := range test.ExpectUIInfo {
					mockUI.EXPECT().Info(expected)
				}

				for _, expected := range test.ExpectUIWarn {
					mockUI.EXPECT().Warn(expected)
				}

				for _, expect := range test.Responses {
					mockUI.EXPECT().Ask(expect.Question).Return(expect.Answer, nil)
				}

				resolvedConfig, err := resolver.ResolveConfig(ctx)
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

func loadTestCases(t *testing.T, path string) []testcase {
	tests := make([]testcase, 1)
	contents, err := ioutil.ReadFile(path)
	assert.NoError(t, err)
	err = yaml.UnmarshalStrict(contents, &tests)
	assert.NoError(t, err)
	return tests
}
