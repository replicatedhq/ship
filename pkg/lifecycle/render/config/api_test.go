package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"strings"

	"github.com/go-test/deep"
	"github.com/replicatedcom/ship/pkg/api"
	_ "github.com/replicatedcom/ship/pkg/lifecycle/render/config/test-cases/api"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type apiTestcase struct {
	Name        string
	Error       bool
	Config      []libyaml.ConfigGroup
	ViperConfig map[string]interface{}   `yaml:"viper_config"`
	Responses   responsesJson            `yaml:"responses"`
	LiveValues  []map[string]interface{} `yaml:"input"`
	State       map[string]interface{}   `yaml:"state"`
}

type responsesJson struct {
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
	ExpectedValue *ValidationError
	ExpectErr     bool

	Name string
}

type configTestCase struct {
	Config        []libyaml.ConfigGroup
	ExpectedValue []*ValidationError
	ExpectErr     bool

	Name string
}

type configItemWhenTestCase struct {
	Config    []libyaml.ConfigGroup
	ExpectErr bool

	Name string
}

func TestAPIResolver(t *testing.T) {
	ctx := context.Background()

	tests := loadAPITestCases(t, filepath.Join("test-cases", "api"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			v := viper.New()
			testLogger := &logger.TestLogger{T: t}

			builderBuilder := &templates.BuilderBuilder{
				Logger: testLogger,
				Viper:  v,
			}

			resolver := &APIConfigRenderer{
				Logger:         testLogger,
				Viper:          v,
				BuilderBuilder: builderBuilder,
			}

			release := &api.Release{
				Spec: api.Spec{
					Config: api.Config{
						V1: test.Config,
					},
				},
			}

			func() {
				if test.LiveValues == nil {
					test.LiveValues = make([]map[string]interface{}, 0)
				}
				if test.State == nil {
					test.State = make(map[string]interface{})
				}

				var resolvedConfig []libyaml.ConfigGroup
				var err error

				if len(test.LiveValues) == 0 {
					resolvedConfig, err = resolver.ResolveConfig(ctx, release, test.State, make(map[string]interface{}))
				} else {
					// simulate multiple inputs
					for _, liveValues := range test.LiveValues {
						resolvedConfig, err = resolver.ResolveConfig(ctx, release, test.State, liveValues)
					}
				}

				if test.Error {
					req.True(err != nil, "Expected this api call to return an error")
				} else {
					req.NoError(err)
				}

				marshalled, err := json.MarshalIndent(resolvedConfig, "", "    ")
				req.NoError(err)

				areSame := areSameJSON(t, marshalled, []byte(test.Responses.JSON))

				var expected []libyaml.ConfigGroup
				err = json.Unmarshal([]byte(test.Responses.JSON), &expected)
				assert.NoError(t, err)
				req.NoError(err)

				diff := deep.Equal(resolvedConfig, expected)

				req.True(areSame, "%v", strings.Join(diff, "\n"))
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

			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			builderBuilder := &templates.BuilderBuilder{
				Logger: testLogger,
				Viper:  v,
			}
			renderer := &APIConfigRenderer{
				Logger:         testLogger,
				Viper:          v,
				BuilderBuilder: builderBuilder,
			}
			output, err := renderer.resolveConfigValuesMap(test.input, groups)
			req.NoError(err)

			req.Equal(test.results, output)
		})
	}
}

func areSameJSON(t *testing.T, s1, s2 []byte) bool {
	var o1 interface{}
	var o2 interface{}

	err := json.Unmarshal(s1, &o1)
	require.NoError(t, err, string(s1))

	err = json.Unmarshal(s2, &o2)
	require.NoError(t, err, string(s2))

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

			val := configGroupIsHidden(test.Config)

			req.Equal(test.ExpectedValue, val)
		})
	}
}

func TestValidateConfigItem(t *testing.T) {
	tests := []configItemRequiredTestCase{
		{
			Config:        &libyaml.ConfigItem{},
			ExpectedValue: (*ValidationError)(nil),
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

			ExpectedValue: &ValidationError{
				Message:   "Config item alpha is required",
				Name:      "alpha",
				ErrorCode: "MISSING_REQUIRED_VALUE",
			},
			Name: "basic fail",
		},
		{
			Config: &libyaml.ConfigItem{

				Name:     "alpha",
				Title:    "alpha value",
				Required: false,
				Value:    "",
				Default:  "",
			},

			ExpectedValue: (*ValidationError)(nil),
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

			ExpectedValue: (*ValidationError)(nil),
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
			ExpectedValue: (*ValidationError)(nil),
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
			ExpectedValue: (*ValidationError)(nil),
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
			ExpectedValue: (*ValidationError)(nil),
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

			ExpectedValue: (*ValidationError)(nil),
			Name:          "pass due to readonly type",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val := validateConfigItem(test.Config)

			req.Equal(test.ExpectedValue, val)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []configTestCase{
		{
			Config:        []libyaml.ConfigGroup{},
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: []*ValidationError{
				{
					Message:   "Config item alpha is required",
					Name:      "alpha",
					ErrorCode: "MISSING_REQUIRED_VALUE",
				},
			},
			Name: "one group one item, required, no value",
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: []*ValidationError{
				{
					Message:   "Config item alpha is required",
					Name:      "alpha",
					ErrorCode: "MISSING_REQUIRED_VALUE",
				},
			},
			Name: "one group one item, required, not hidden, no value",
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: ([]*ValidationError)(nil),
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
			ExpectedValue: []*ValidationError{
				{
					Message:   "Config item alpha is required",
					Name:      "alpha",
					ErrorCode: "MISSING_REQUIRED_VALUE",
				},
				{
					Message:   "Config item beta is required",
					Name:      "beta",
					ErrorCode: "MISSING_REQUIRED_VALUE",
				},
			},
			Name: "one group two items, required",
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
			ExpectedValue: []*ValidationError{
				{
					Message:   "Config item beta is required",
					Name:      "beta",
					ErrorCode: "MISSING_REQUIRED_VALUE",
				},
			},
			Name: "one group two items, required",
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
			ExpectedValue: ([]*ValidationError)(nil),
			Name:          "one group two items, required",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			val := validateConfig(test.Config)

			req.Equal(test.ExpectedValue, val)
		})
	}
}

func TestWhenConfigItem(t *testing.T) {
	tests := []configItemWhenTestCase{
		{
			Config: []libyaml.ConfigGroup{},
			Name:   "empty test",
		},
		{
			Config: []libyaml.ConfigGroup{
				{
					Name: "testing",
					Items: []*libyaml.ConfigItem{
						{
							Name:  "alpha",
							Value: "1",
						},
						{
							Name: "beta",
							Type: "bool",
							When: `{{repl ConfigOptionEquals "alpha" "1" }}`,
						},
					},
				},
			},
			Name: "type bool",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

		})
	}
}
