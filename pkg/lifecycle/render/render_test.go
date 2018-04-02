package render

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/gojuno/minimock"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/plan"
	"github.com/replicatedcom/ship/pkg/test-fixtures/config"
	"github.com/replicatedcom/ship/pkg/test-fixtures/planner"
	"github.com/replicatedcom/ship/pkg/test-fixtures/ui"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

type testcase struct {
	Name        string
	Spec        *api.Spec
	ViperConfig map[string]interface{} `yaml:"viper_config"`
	Responses   map[string]string
	Expect      map[string]string
}

func TestRender(t *testing.T) {
	ctx := context.Background()

	renderer := &Renderer{
		Logger: log.NewNopLogger(),
	}

	tests := loadTestCases(t, filepath.Join("test-fixtures", "render-inline.yaml"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mc := minimock.NewController(t)
			mockUI := ui.NewUiMock(mc)
			p := planner.NewIPlannerMock(mc)
			configResolver := config.NewIResolverMock(mc)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}

			renderer.Spec = test.Spec
			renderer.Fs = mockFS
			renderer.UI = mockUI
			renderer.ConfigResolver = configResolver
			renderer.Planner = p

			func() {
				defer mc.Finish()

				configResolver.ResolveConfigMock.
					Expect(ctx).
					Return(test.ViperConfig, nil)

				p.BuildMock.
					Expect(test.Spec.Assets.V1, test.ViperConfig).
					Return(plan.Plan{})

				p.ExecuteMock.
					Expect(ctx, plan.Plan{}).
					Return(nil)

				p.ConfirmMock.Expect(plan.Plan{}).Return(true, nil)

				// todo test state ops

				err := renderer.Execute(ctx, &api.Render{})
				assert.NoError(t, err)
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
