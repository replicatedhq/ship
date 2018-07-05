package render

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	_ "github.com/replicatedhq/ship/pkg/lifecycle/render/test-cases"

	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	mockconfig "github.com/replicatedhq/ship/pkg/test-mocks/config"
	mockplanner "github.com/replicatedhq/ship/pkg/test-mocks/planner"
	"github.com/replicatedhq/ship/pkg/test-mocks/ui"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type testcase struct {
	Name        string
	Metadata    api.ReleaseMetadata
	Spec        api.Spec
	ViperConfig map[string]interface{} `yaml:"viper_config"`
	Responses   map[string]string
	Expect      map[string]string
}

func TestRender(t *testing.T) {
	ctx := context.Background()

	tests := loadTestCases(t, filepath.Join("test-cases", "render-inline.yaml"))

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mc := gomock.NewController(t)

			mockUI := ui.NewMockUi(mc)
			p := mockplanner.NewMockPlanner(mc)
			configResolver := mockconfig.NewMockResolver(mc)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			mockDaemon := mockconfig.NewMockDaemon(mc)

			renderer := &Renderer{
				Logger: log.NewNopLogger(),
				Now:    time.Now,
			}
			renderer.Fs = mockFS
			renderer.UI = mockUI
			renderer.ConfigResolver = configResolver
			renderer.Planner = p
			renderer.StateManager = &state.Manager{
				Logger: renderer.Logger,
				FS:     mockFS,
				V:      viper.New(),
			}

			prog := mockDaemon.EXPECT().SetProgress(ProgressLoad)
			prog = mockDaemon.EXPECT().SetProgress(ProgressResolve).After(prog)
			prog = mockDaemon.EXPECT().SetProgress(ProgressBuild).After(prog)
			prog = mockDaemon.EXPECT().SetProgress(ProgressBackup).After(prog)
			prog = mockDaemon.EXPECT().SetProgress(ProgressExecute).After(prog)
			prog = mockDaemon.EXPECT().SetStepName(ctx, config.StepNameConfirm).After(prog)
			prog = mockDaemon.EXPECT().SetProgress(ProgressCommit).After(prog)
			mockDaemon.EXPECT().ClearProgress().After(prog)

			p.EXPECT().WithDaemon(mockDaemon).Return(p)
			configResolver.EXPECT().WithDaemon(mockDaemon).Return(configResolver)

			renderer = renderer.WithDaemon(mockDaemon)

			release := &api.Release{Spec: test.Spec}

			func() {
				defer mc.Finish()

				configResolver.EXPECT().
					ResolveConfig(ctx, release, gomock.Any()).
					Return(test.ViperConfig, nil)

				p.EXPECT().
					Build(test.Spec.Assets.V1, test.Spec.Config.V1, gomock.Any(), test.ViperConfig).
					Return(planner.Plan{}, nil)

				p.EXPECT().
					Execute(ctx, planner.Plan{}).
					Return(nil)

				err := renderer.Execute(ctx, release, &api.Render{})
				assert.NoError(t, err)
			}()
		})
	}
}

func TestBacksUpExisting(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		existing []string
		expect   []string
	}{
		{
			name:     "first",
			target:   "/tmp/installer",
			existing: []string{},
			expect:   []string{},
		},
		{
			name:   "first",
			target: "/tmp/installer",
			existing: []string{
				"/tmp/installer",
			},
			expect: []string{
				"/tmp/installer.bak",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			r := Renderer{
				Logger: &logger.TestLogger{T: t},
				Fs:     mockFS,
				Now: func() time.Time {
					return time.Unix(12345, 0)
				},
			}

			for _, filename := range test.existing {
				err := mockFS.WriteFile(filename, []byte("not a directory but thats okay"), 0755)
				req.NoError(err)
			}

			r.backupIfPresent(test.target)

			debugFs := &strings.Builder{}
			r.Fs.Walk("/", func(path string, info os.FileInfo, err error) error {
				debugFs.WriteString(path)
				debugFs.WriteString("\n")
				return nil
			})

			for _, filename := range test.expect {
				exists, err := mockFS.Exists(filename)
				req.NoError(err)
				req.True(exists, "expected file %s to exist, fs had %s", filename, debugFs)
			}

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
