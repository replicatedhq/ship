package render

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

type testcase struct {
	Name        string
	Spec        *api.Spec
	ViperConfig map[string]string `yaml:"viper_config"`
	Expect      map[string]string
}

func TestRender(t *testing.T) {
	ctx := context.Background()
	mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
	mockUi := cli.NewMockUi()
	mockViper := viper.New()

	step := &Renderer{
		Step: &api.Render{
			SkipPlan: true,
		},
		Fs:     mockFS,
		UI:     mockUi,
		Logger: log.NewNopLogger(),
		Viper:  mockViper,
	}

	path := filepath.Join("test-fixtures", "render-inline.yaml")
	tests := loadFixtureData(t, path)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			step.Spec = test.Spec

			for key, value := range test.ViperConfig {
				mockViper.Set(key, value)
			}

			err := step.Execute(ctx)
			assert.Nil(t, err)

			for path, expected := range test.Expect {
				contents, err := mockFS.ReadFile(path)
				assert.Nil(t, err)
				assert.Equal(t, expected, string(contents))
			}

		})
	}
}
func loadFixtureData(t *testing.T, path string) []testcase {
	tests := make([]testcase, 1)
	contents, err := ioutil.ReadFile(path)
	assert.Nil(t, err)
	err = yaml.UnmarshalStrict(contents, &tests)
	assert.Nil(t, err)
	return tests
}
