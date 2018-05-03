package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type apiTestcase struct {
	Name        string
	Config      []libyaml.ConfigGroup
	ViperConfig map[string]interface{} `yaml:"viper_config"`
	Responses   apiExpectUIAsk         `yaml:"responses"`
	Expect      map[string]string
}

type apiExpectUIAsk struct {
	JSON string `yaml:"json"`
}

func TestAPIResolver(t *testing.T) {
	ctx := context.Background()

	resolver := &APIResolver{
		Logger: log.NewNopLogger(),
	}

	tests := loadAPITestCases(t, filepath.Join("test-cases", "api"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			resolver.Release = &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: test.Config,
					},
				},
			}
			resolver.Viper = viper.New()

			func() {
				resolvedConfig, err := resolver.ResolveConfig(ctx, nil, make(map[string]interface{}))
				req.NoError(err)

				marshalled, err := json.Marshal(resolvedConfig)
				req.NoError(err)

				areSame := areSameJSON(t, marshalled, []byte(test.Responses.JSON))
				req.True(areSame, "%s should be %s", marshalled, test.Responses.JSON)
			}()
		})
	}
}

func areSameJSON(t *testing.T, s1, s2 []byte) bool {
	var o1 interface{}
	var o2 interface{}

	err := json.Unmarshal(s1, &o1)
	assert.NoError(t, err)

	err = json.Unmarshal(s2, &o2)
	assert.NoError(t, err)

	return reflect.DeepEqual(o1, o2)
}

func loadAPITestCases(t *testing.T, path string) []apiTestcase {
	files, err := ioutil.ReadDir(path)
	assert.NoError(t, err)

	tests := make([]apiTestcase, 0)

	for _, file := range files {
		if filepath.Ext(filepath.Join(path, file.Name())) != ".yml" {
			continue
		}

		contents, err := ioutil.ReadFile(filepath.Join(path, file.Name()))
		assert.NoError(t, err)

		test := make([]apiTestcase, 0)
		err = yaml.UnmarshalStrict(contents, &test)
		assert.NoError(t, err)

		tests = append(tests, test...)
	}

	return tests
}
