package helm

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/logger"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	tests := []struct {
		name        string
		asset       api.HelmAsset
		expect      string
		expectError string
	}{
		{
			name: "nil local fails",
			asset: api.HelmAsset{
				Local: nil,
			},
			expect:      "",
			expectError: "only 'local' chart rendering is supported",
		},
		{
			name: "local returns pre-configured location",
			asset: api.HelmAsset{
				Local: &api.LocalHelmOpts{
					ChartRoot: "charts/nginx",
				},
			},
			expect:      "installer/charts/nginx",
			expectError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			fetcher := &ClientFetcher{
				Logger: &logger.TestLogger{T: t},
			}

			dest, err := fetcher.FetchChart(test.asset, api.ReleaseMetadata{})

			if test.expectError == "" {
				req.NoError(err)
			} else {
				req.Error(err, "expected error "+test.expectError)
				req.Equal(test.expectError, err.Error())
			}

			req.Equal(test.expect, dest)
		})
	}
}
