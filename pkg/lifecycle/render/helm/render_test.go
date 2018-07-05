package helm

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/test-mocks/helm"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name        string
		fetchPath   string
		fetchErr    error
		templateErr error
		expectErr   error
	}{
		{
			name:        "fetch fails",
			fetchPath:   "/etc/kfbr",
			fetchErr:    errors.New("fetch failed"),
			templateErr: nil,
			expectErr:   errors.New("fetch chart: fetch failed"),
		},
		{
			name:        "template fails",
			fetchPath:   "/etc/kfbr",
			fetchErr:    nil,
			templateErr: errors.New("template failed"),
			expectErr:   errors.New("execute templating: template failed"),
		},
		{
			name:        "all good",
			fetchPath:   "/etc/kfbr",
			fetchErr:    nil,
			templateErr: nil,
			expectErr:   nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			mockFetcher := helm.NewMockChartFetcher(mc)
			mockTemplater := helm.NewMockTemplater(mc)
			req := require.New(t)
			renderer := &LocalRenderer{
				Fetcher:   mockFetcher,
				Templater: mockTemplater,
			}

			asset := api.HelmAsset{}
			metadata := api.ReleaseMetadata{}
			templateContext := map[string]interface{}{}
			configGroups := []libyaml.ConfigGroup{}

			ctx := context.Background()

			mockFetcher.EXPECT().
				FetchChart(ctx, asset, metadata, configGroups, templateContext).
				Return(test.fetchPath, test.fetchErr)

			if test.fetchErr == nil {
				mockTemplater.EXPECT().
					Template(test.fetchPath, asset, metadata, configGroups, templateContext).
					Return(test.templateErr)
			}

			err := renderer.Execute(
				asset,
				metadata,
				templateContext,
				configGroups,
			)(ctx)

			if test.expectErr == nil {
				req.NoError(err)
			} else {
				req.Error(err, "expected error "+test.expectErr.Error())
				req.Equal(test.expectErr.Error(), err.Error())
			}

		})
	}
}
