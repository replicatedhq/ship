package ship

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/stretchr/testify/require"
)

type CheckUpstreamRelease struct {
	Name        string
	Description string
	Expected    *api.Lifecycle
}

type ApplyUpstreamReleaseLifecycle struct {
	Name              string
	Description       string
	UpstreamLifecycle *api.Lifecycle
	DefaultLifecycle  *api.Lifecycle
	UpstreamExists    bool
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

func TestApplyUpstreamReleaseLifecycle(t *testing.T) {
	tests := []ApplyUpstreamReleaseLifecycle{
		{
			Name:              "no upstream",
			Description:       "no upstream, should use default release lifecycle",
			UpstreamLifecycle: nil,
			DefaultLifecycle:  nil,
			UpstreamExists:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			s := &Ship{}
			meta := api.HelmChartMetadata{}

			var release *api.Release
			if test.UpstreamExists {
				release = s.buildHelmRelease(meta, test.UpstreamLifecycle)
				req.Equal(release.Spec.Lifecycle, test.UpstreamLifecycle)
			} else {
				release = s.buildHelmRelease(meta, DefaultHelmLifecycle)
				req.Equal(release.Spec.Lifecycle, test.DefaultLifecycle)
			}
		})
	}
}
