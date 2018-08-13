package ship

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/stretchr/testify/require"
)

type ApplyUpstreamReleaseSpec struct {
	Name            string
	Description     string
	UpstreamRelease *api.Release
	DefaultRelease  *api.Release
}

func TestApplyUpstreamReleaseSpec(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:            "no upstream",
			Description:     "no upstream, should use default release spec",
			UpstreamRelease: &api.Release{},
			DefaultRelease:  DefaultHelmRelease,
		},
		{
			Name:            "upstream exists",
			Description:     "upstream exists, should use upstream release spec",
			UpstreamRelease: DefaultHelmRelease,
			DefaultRelease:  DefaultHelmRelease,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			s := &Ship{}
			meta := api.HelmChartMetadata{}

			release := s.buildHelmRelease(meta, *test.UpstreamRelease)

			req.Equal(release, test.DefaultRelease)
		})
	}
}
