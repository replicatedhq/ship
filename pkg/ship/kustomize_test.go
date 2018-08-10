package ship

import (
	"path/filepath"
	"testing"

	"encoding/json"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type CheckUpstreamRelease struct {
	Name         string
	Description  string
	ExpectedSpec *api.Spec
	ExpectedErr  bool
}

type ApplyUpstreamReleaseSpec struct {
	Name           string
	Description    string
	UpstreamSpec   *api.Spec
	DefaultSpec    *api.Spec
	UpstreamExists bool
}

func TestCheckUpstreamRelease(t *testing.T) {
	tests := []CheckUpstreamRelease{
		{
			Name:         "no upstream",
			Description:  "no upstream, should return nil",
			ExpectedSpec: nil,
			ExpectedErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			s := &Ship{}
			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}

			if test.ExpectedSpec != nil {
				upstreamRelease, err := json.Marshal(test.ExpectedSpec)
				req.NoError(err)

				err = fakeFS.WriteFile(filepath.Join(constants.KustomizeHelmPath, "ship.yaml"), upstreamRelease, 0666)
				req.NoError(err)
			}

			upstreamRelease, err := s.checkUpstreamForRelease()

			if test.ExpectedErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			req.Equal(upstreamRelease, test.ExpectedSpec)
		})
	}
}

func TestApplyUpstreamReleaseSpec(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:           "no upstream",
			Description:    "no upstream, should use default release spec",
			UpstreamSpec:   nil,
			DefaultSpec:    DefaultHelmSpec,
			UpstreamExists: false,
		},
		{
			Name:           "empty upstream",
			Description:    "empty upstream exists, should use upstream release spec",
			UpstreamSpec:   &api.Spec{},
			DefaultSpec:    DefaultHelmSpec,
			UpstreamExists: true,
		},
		{
			Name:        "empty upstream with fields",
			Description: "empty upstream with fields exists, should use upstream release spec",
			UpstreamSpec: &api.Spec{
				Assets:    api.Assets{},
				Lifecycle: api.Lifecycle{},
			},
			DefaultSpec:    DefaultHelmSpec,
			UpstreamExists: true,
		},
		{
			Name:           "large upstream",
			Description:    "large upstream exists, should use upstream release spec",
			UpstreamSpec:   DefaultHelmSpec,
			DefaultSpec:    DefaultHelmSpec,
			UpstreamExists: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			s := &Ship{}
			meta := api.HelmChartMetadata{}

			var release *api.Release
			if test.UpstreamExists {
				release = s.buildHelmRelease(meta, test.UpstreamSpec)
				req.Equal(&release.Spec, test.UpstreamSpec)
			} else {
				release = s.buildHelmRelease(meta, DefaultHelmSpec)
				req.Equal(&release.Spec, test.DefaultSpec)
			}
		})
	}
}
