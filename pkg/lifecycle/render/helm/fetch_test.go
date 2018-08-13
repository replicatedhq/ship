package helm

import (
	"testing"

	"context"

	"strings"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/test-mocks/github"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	tests := []struct {
		name        string
		asset       api.HelmAsset
		expect      string
		mockExpect  func(t *testing.T, gh *github.MockRenderer)
		expectError string
	}{
		{
			name: "nil local fails",
			asset: api.HelmAsset{
				Local:  nil,
				GitHub: nil,
			},
			expect:      "",
			expectError: "only 'local' and 'github' chart rendering is supported",
		},
		{
			name: "local returns location",
			asset: api.HelmAsset{
				Local: &api.LocalHelmOpts{
					ChartRoot: "charts/nginx",
				},
			},
			expect:      "charts/nginx",
			expectError: "",
		},
		{
			name: "github fetches from github",
			asset: api.HelmAsset{
				GitHub: &api.GitHubAsset{
					Ref:    "",
					Repo:   "",
					Path:   "",
					Source: "",
				},
			},
			mockExpect: func(t *testing.T, gh *github.MockRenderer) {
				gh.EXPECT().Execute(
					root.Fs{
						Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
						RootPath: "",
					},
					&matchers.Is{
						Describe: "is github asset and has dest overriden",
						Test: func(asset interface{}) bool {
							githubAsset, ok := asset.(api.GitHubAsset)
							if !ok {
								return false
							}
							return strings.HasPrefix(githubAsset.Dest, "/tmp/helmchart")

						},
					},
					[]libyaml.ConfigGroup{},
					api.ReleaseMetadata{},
					map[string]interface{}{},
				).Return(func(ctx context.Context) error { return nil })
			},
			expect:      "/tmp/helmchart",
			expectError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			req := require.New(t)
			gh := github.NewMockRenderer(mc)
			mockfs := afero.Afero{Fs: afero.NewMemMapFs()}
			fetcher := &ClientFetcher{
				Logger: &logger.TestLogger{T: t},
				GitHub: gh,
				FS:     mockfs,
			}

			if test.mockExpect != nil {
				test.mockExpect(t, gh)
			}

			dest, err := fetcher.FetchChart(
				context.Background(),
				root.Fs{
					Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
					RootPath: "",
				},
				test.asset,
				api.ReleaseMetadata{},
				[]libyaml.ConfigGroup{},
				map[string]interface{}{},
			)

			if test.expectError == "" {
				req.NoError(err)
			} else {
				req.Error(err, "expected error "+test.expectError)
				req.Equal(test.expectError, err.Error())
			}

			req.True(
				strings.HasPrefix(dest, test.expect),
				"expected %s to have prefix %s",
				dest,
				test.expect,
			)
		})
	}
}
