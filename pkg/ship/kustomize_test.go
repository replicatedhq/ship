package ship

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/stretchr/testify/require"
)

type CheckUpstreamRelease struct {
	Name        string
	Description string
	Expected    *api.Spec
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
			Name:        "no upstream",
			Description: "no upstream, should return nil",
			Expected:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			s := &Ship{}

			// todo where is upstream?

			upstreamRelease := s.checkUpstreamForRelease()

			req.Equal(upstreamRelease, test.Expected)
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
