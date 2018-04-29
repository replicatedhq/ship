package config

import (
	"context"
	"encoding/json"
	"fmt"
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

	tests := loadAPITestCases(t, filepath.Join("test-fixtures", "config-test-api.yml"))

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
				resolvedConfig, err := resolver.ResolveConfig(ctx, nil)
				req.NoError(err)

				marshalled, err := json.Marshal(resolvedConfig)
				req.NoError(err)

				areSame, err := areSameJSON(marshalled, []byte(test.Responses.JSON))
				req.NoError(err)
				req.True(areSame, "%s should be %s", marshalled, test.Responses.JSON)
			}()
		})
	}
}

func areSameJSON(s1, s2 []byte) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal(s1, &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal(s2, &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}

func loadAPITestCases(t *testing.T, path string) []apiTestcase {
	tests := make([]apiTestcase, 1)
	contents, err := ioutil.ReadFile(path)
	assert.NoError(t, err)
	err = yaml.UnmarshalStrict(contents, &tests)
	assert.NoError(t, err)
	return tests
}
