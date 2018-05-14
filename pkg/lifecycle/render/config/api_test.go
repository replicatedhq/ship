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
	_ "github.com/replicatedcom/ship/pkg/lifecycle/render/config/test-cases/api"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type apiTestcase struct {
	Name        string
	Error       bool
	Config      []libyaml.ConfigGroup
	ViperConfig map[string]interface{} `yaml:"viper_config"`
	Responses   apiExpectUIAsk         `yaml:"responses"`
	Input       map[string]interface{} `yaml:"input"`
}

type apiExpectUIAsk struct {
	JSON string `yaml:"json"`
}

type configValuesTestCase struct {
	dependencies map[string][]string
	input        map[string]interface{}
	results      map[string]interface{}
	prefix       string
	suffix       string

	Name string
}

type configRequiredTestCase struct {
	Config        []libyaml.ConfigGroup
	ExpectedValue bool
	ExpectErr     bool

	Name string
}

func TestAPIResolver(t *testing.T) {
	ctx := context.Background()

	resolver := &APIConfigRenderer{
		Logger: log.NewNopLogger(),
	}

	tests := loadAPITestCases(t, filepath.Join("test-cases", "api"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			release := &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: test.Config,
					},
				},
			}
			resolver.Viper = viper.New()

			func() {
				if test.Input == nil {
					test.Input = make(map[string]interface{})
				}
				resolvedConfig, err := resolver.ResolveConfig(ctx, release, test.Input)
				if test.Error {
					req.True(err != nil, "Expected this api call to return an error")
				} else {
					req.NoError(err)
				}

				marshalled, err := json.MarshalIndent(resolvedConfig, "", "    ")
				req.NoError(err)

				areSame := areSameJSON(t, marshalled, []byte(test.Responses.JSON))
				req.True(areSame, "%s\nand should be \n%s", marshalled, test.Responses.JSON)
			}()
		})
	}
}

func TestResolveConfigValuesMap(t *testing.T) {
	tests := []configValuesTestCase{
		{
			dependencies: map[string][]string{
				"alpha": {},
				"bravo": {"alpha"},
			},
			input:   map[string]interface{}{"alpha": "abc"},
			results: map[string]interface{}{"alpha": "abc", "bravo": "abc"},
			Name:    "basic_dependency",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"bravo"},
			},
			input:   map[string]interface{}{"alpha": "abc"},
			results: map[string]interface{}{"alpha": "abc", "bravo": "(abc)", "charlie": "((abc))"},
			prefix:  "(",
			suffix:  ")",
			Name:    "basic_chain",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {},
				"charlie": {"alpha", "bravo"},
			},
			input:   map[string]interface{}{"alpha": "abc", "bravo": "xyz"},
			results: map[string]interface{}{"alpha": "abc", "bravo": "xyz", "charlie": "(abcxyz)"},
			prefix:  "(",
			suffix:  ")",
			Name:    "basic_2deps",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {},
				"charlie": {"alpha", "bravo"},
				"delta":   {"charlie"},
			},
			input:   map[string]interface{}{"alpha": "abc", "bravo": "xyz"},
			results: map[string]interface{}{"alpha": "abc", "bravo": "xyz", "charlie": "(abcxyz)", "delta": "((abcxyz))"},
			prefix:  "(",
			suffix:  ")",
			Name:    "basic_Y_shape",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"alpha"},
				"delta":   {"bravo", "charlie"},
			},
			input:   map[string]interface{}{"alpha": "abc"},
			results: map[string]interface{}{"alpha": "abc", "bravo": "(abc)", "charlie": "(abc)", "delta": "((abc)(abc))"},
			prefix:  "(",
			suffix:  ")",
			Name:    "basic_◇_shape",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			//build a config to test
			groups := buildTestConfigGroups(test.dependencies, test.prefix, test.suffix, false)

			output, err := resolveConfigValuesMap(test.input, groups, log.NewNopLogger(), viper.New())
			req.NoError(err)

			req.Equal(test.results, output)
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

func TestValidateConfig(t *testing.T) {
	tests := []configRequiredTestCase{
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
							Required: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedValue: true,
			Name:          "basic fail",
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
			Name:          "basic pass",
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
							Value:    "value",
							Default:  "",
						},
					},
				},
			},
			ExpectedValue: false,
			Name:          "pass due to value",
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
							Default:  "default",
						},
					},
				},
			},
			ExpectedValue: false,
			Name:          "pass due to default",
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
							Hidden:   true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedValue: false,
			Name:          "pass due to hidden",
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
							ReadOnly: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedValue: false,
			Name:          "pass due to readonly set",
		},
		{
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:     "alpha",
							Title:    "alpha value",
							Type:     "label",
							Required: true,
							Value:    "",
							Default:  "",
						},
					},
				},
			},
			ExpectedValue: false,
			Name:          "pass due to readonly type",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val, err := ValidateConfig(context.Background(), test.Config)
			if test.ExpectErr {
				req.Error(err)
				return
			} else {
				req.NoError(err)
			}

			req.Equal(test.ExpectedValue, val)
		})
	}
}
