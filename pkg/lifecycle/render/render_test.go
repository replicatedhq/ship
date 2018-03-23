package render

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/gojuno/minimock"
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
	Responses   map[string]string
	Expect      map[string]string
	ExpectInfo  []string `yaml:"expect_info"`
}

func TestRender(t *testing.T) {
	ctx := context.Background()
	mockFS := afero.Afero{Fs: afero.NewMemMapFs()}

	renderer := &Renderer{
		Fs:     mockFS,
		Logger: log.NewNopLogger(),
	}

	tests := loadTestCases(t, filepath.Join("test-fixtures", "render-inline.yaml"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mc := minimock.NewController(t)
			mockUI := NewUiMock(mc)
			mockViper := viper.New()
			renderer.Spec = test.Spec

			renderer.ConfigResolver = &ConfigResolver{
				Fs:     renderer.Fs,
				Logger: renderer.Logger,
				Spec:   renderer.Spec,
				UI:     mockUI,
				Viper:  mockViper,
			}

			func() {
				defer mc.Finish()

				for ask, response := range test.Responses {
					mockUI.AskMock.Expect(ask).Return(response, nil)
				}

				for _, info := range test.ExpectInfo {
					mockUI.InfoMock.Expect(info).Return()
				}

				for key, value := range test.ViperConfig {
					mockViper.Set(key, value)
				}

				err := renderer.Execute(ctx, &api.Render{})
				assert.NoError(t, err)

				for path, expected := range test.Expect {
					contents, err := mockFS.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, expected, string(contents))
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
