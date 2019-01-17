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

var _ = Describe("specs.Resolver", func() {

	Describe("calculateContentSHA", func() {
		Context("With multiple files", func() {
			It("should calculate the same sha, multiple times", func() {
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
				mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)

				r := Resolver{
					FS: mockFs,
					StateManager: &state.MManager{
						Logger: log.NewNopLogger(),
						FS:     mockFs,
						V:      viper.New(),
					},
				}

				firstPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				secondPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				Expect(firstPass).To(Equal(secondPass))
			})
		})
	})
})

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
				mockFs.WriteFile(filepath.Join(constants.HelmChartPath, "ship.yaml"), []byte(test.UpstreamShipYAML), 0755)
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
