package terraform

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/test-mocks/inline"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestRenderer(t *testing.T) {
	tests := []struct {
		name  string
		asset api.TerraformAsset
	}{
		{
			name: "empty",
			asset: api.TerraformAsset{
				Inline: "some tf config",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockInline := inline.NewMockRenderer(mc)

			renderer := &LocalRenderer{
				Logger: &logger.TestLogger{T: t},
				Inline: mockInline,
			}

			assetMatcher := &matchers.Is{
				Describe: "inline asset",
				Test: func(v interface{}) bool {
					asset, ok := v.(api.InlineAsset)
					if !ok {
						return false
					}
					return asset.Contents == test.asset.Inline
				},
			}

			rootFs := root.Fs{
				Afero: afero.Afero{
					Fs: afero.NewBasePathFs(afero.NewMemMapFs(), constants.InstallerPrefixPath),
				},
				RootPath: constants.InstallerPrefixPath,
			}
			metadata := api.ReleaseMetadata{}
			groups := []libyaml.ConfigGroup{}
			templateContext := map[string]interface{}{}

			mockInline.EXPECT().Execute(
				rootFs,
				assetMatcher,
				metadata,
				templateContext,
				groups,
			).Return(func(ctx context.Context) error { return nil })

			err := renderer.Execute(
				rootFs,
				test.asset,
				metadata,
				templateContext,
				groups,
			)(context.Background())

			req.NoError(err)
		})
	}
}
