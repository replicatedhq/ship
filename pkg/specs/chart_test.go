package specs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type ApplyUpstreamReleaseSpec struct {
	Name             string
	Description      string
	UpstreamShipYAML string
	ExpectedSpec     *api.Spec
}

func TestSpecsResolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "specsResolver")
}

// the same content should result in the same hash when hashed twice
func TestStableContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	req.Equal(firstPass, secondPass)
}

// different content should not result in the same hash
func TestChangingContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	err = mockFs.WriteFile("newfile.txt", []byte("I AM A NEW FILE"), 0755)
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	req.NotEqual(firstPass, secondPass)
}

// content sha should not depend on the contents of a `.git` directory
func TestStableGitContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile(".git/A_FILE", []byte("A_GIT_FILE"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	err = mockFs.WriteFile(".git/newfile.txt", []byte("I AM A NEW GIT FILE"), 0755)
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	req.Equal(firstPass, secondPass)
}

// content sha should not depend on the root path
func TestRootIndependentContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("path1/Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("path1/templates/README.md", []byte("readme"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("path2/Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("path2/templates/README.md", []byte("readme"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "path1")
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "path2")
	req.NoError(err)

	req.Equal(firstPass, secondPass)
}

// content sha should change if a filename changes
func TestMoveFileContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	err = mockFs.Rename("templates/README.md", "templated/README.md")
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	req.NotEqual(firstPass, secondPass)
}

// content sha should change if a filename changes
func TestEditFileContentSha(t *testing.T) {
	req := require.New(t)
	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	err := mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)
	req.NoError(err)

	r := Resolver{
		FS: mockFs,
		StateManager: &state.MManager{
			Logger: log.NewNopLogger(),
			FS:     mockFs,
			V:      viper.New(),
		},
	}

	firstPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	err = mockFs.Remove("templates/README.md")
	req.NoError(err)
	err = mockFs.WriteFile("templates/README.md", []byte("not readme"), 0755)
	req.NoError(err)

	secondPass, err := calculateContentSHA(r.FS, "")
	req.NoError(err)

	req.NotEqual(firstPass, secondPass)
}

func TestMaybeGetShipYAML(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:         "no upstream",
			Description:  "no upstream, should use default release spec",
			ExpectedSpec: nil,
		},
		{
			Name:        "upstream exists",
			Description: "upstream exists, should use upstream release spec",
			UpstreamShipYAML: `
assets:
  v1: []
config:
  v1: []
lifecycle:
  v1:
   - helmIntro: {}
`,
			ExpectedSpec: &api.Spec{
				Assets: api.Assets{
					V1: []api.Asset{},
				},
				Config: api.Config{
					V1: []libyaml.ConfigGroup{},
				},
				Lifecycle: api.Lifecycle{
					V1: []api.Step{
						{
							HelmIntro: &api.HelmIntro{},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			if test.UpstreamShipYAML != "" {
				err := mockFs.WriteFile(filepath.Join(constants.HelmChartPath, "ship.yaml"), []byte(test.UpstreamShipYAML), 0755)
				req.NoError(err)
			}

			r := Resolver{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
				ui:     cli.NewMockUi(),
			}

			ctx := context.Background()
			spec, err := r.maybeGetShipYAML(ctx, constants.HelmChartPath)
			req.NoError(err)

			req.Equal(test.ExpectedSpec, spec)
		})
	}
}
