package ship

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/stretchr/testify/require"
)

type CheckUpstreamRelease struct {
	Name        string
	Description string
	Expected    *api.Release
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

			upstreamRelease := s.checkUpstreamForRelease()

			req.Equal(upstreamRelease, test.Expected)
		})
	}
}

func TestApplyUpstreamReleaseSpec(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:           "no upstream",
			Description:    "no upstream, should use default release lifecycle",
			UpstreamSpec:   nil,
			DefaultSpec:    DefaultHelmSpec,
			UpstreamExists: false,
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
				req.Equal(release.Spec, test.UpstreamSpec)
			} else {
				release = s.buildHelmRelease(meta, DefaultHelmSpec)
				req.Equal(release.Spec, test.DefaultSpec)
			}
		})
	}
}
