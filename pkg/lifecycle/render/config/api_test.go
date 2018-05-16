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

type configGroupHiddenTestCase struct {
	Config        libyaml.ConfigGroup
	ExpectedValue bool
	ExpectErr     bool

	Name string
}

type configItemRequiredTestCase struct {
	Config        *libyaml.ConfigItem
	ExpectedValue string
	ExpectErr     bool

	Name string
}
type configTestCase struct {
	Config        []libyaml.ConfigGroup
	ExpectedValue []string
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

func TestHiddenConfigGroup(t *testing.T) {
	tests := []configGroupHiddenTestCase{
		{
			Config:        libyaml.ConfigGroup{},
			ExpectedValue: true,
			Name:          "empty test",
		},
		{
			Config: libyaml.ConfigGroup{
				Name: "one hidden item = hidden group",
				Items: []*libyaml.ConfigItem{
					{
						Name:   "alpha",
						Hidden: true,
					},
				},
			},
			ExpectedValue: true,
			Name:          "one item, hidden => hidden group",
		},
		{
			Config: libyaml.ConfigGroup{
				Name: "two hidden items = hidden group",
				Items: []*libyaml.ConfigItem{
					{
						Name:   "alpha",
						Hidden: true,
					},
					{
						Name:   "beta",
						Hidden: true,
					},
				},
			},
			ExpectedValue: true,
			Name:          "two items, both hidden => hidden group",
		},
		{
			Config: libyaml.ConfigGroup{
				Name: "one item, not hidden",
				Items: []*libyaml.ConfigItem{
					{
						Name:   "alpha",
						Hidden: false,
					},
				},
			},
			ExpectedValue: false,
			Name:          "one item, not hidden => NOT hidden group",
		},
		{
			Config: libyaml.ConfigGroup{
				Name: "two items, one hidden",
				Items: []*libyaml.ConfigItem{
					{
						Name:   "alpha",
						Hidden: true,
					},
					{
						Name:   "beta",
						Hidden: false,
					},
				},
			},
			ExpectedValue: false,
			Name:          "two items, one hidden => NOT hidden group",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val, err := configGroupIsHidden(test.Config)
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

func TestValidateConfigItem(t *testing.T) {
	tests := []configItemRequiredTestCase{
		{
			Config:        &libyaml.ConfigItem{},
			ExpectedValue: "",
			Name:          "empty test",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: true,
				Value:    "",
				Default:  "",
			},

			ExpectedValue: "Config item alpha is required",
			Name:          "basic fail",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: false,
				Value:    "",
				Default:  "",
			},

			ExpectedValue: "",
			Name:          "basic pass",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: true,
				Value:    "value",
				Default:  "",
			},

			ExpectedValue: "",
			Name:          "pass due to value",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: true,
				Value:    "",
				Default:  "default",
			},
			ExpectedValue: "",
			Name:          "pass due to default",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: true,
				Hidden:   true,
				Value:    "",
				Default:  "",
			},
			ExpectedValue: "",
			Name:          "pass due to hidden",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: true,
				ReadOnly: true,
				Value:    "",
				Default:  "",
			},
			ExpectedValue: "",
			Name:          "pass due to readonly set",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Type:     "label",
				Required: true,
				Value:    "",
				Default:  "",
			},

			ExpectedValue: "",
			Name:          "pass due to readonly type",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val, err := validateConfigItem(test.Config)
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

func TestValidateConfig(t *testing.T) {
	tests := []configTestCase{
		{
			Config:        []libyaml.ConfigGroup{},
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string{"Config item alpha is required"},
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string{"Config item alpha is required"},
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string(nil),
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
			ExpectedValue: []string{"Config item alpha is required", "Config item beta is required"},
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
			ExpectedValue: []string{"Config item beta is required"},
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
			ExpectedValue: []string(nil),
			Name:          "one group two items, required",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val, err := validateConfig(test.Config)
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
