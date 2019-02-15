package helm

import (
	"context"
	"strings"
	"testing"

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
		renderRoot  string
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
			expectError: "only 'local', 'github' and 'helm_fetch' chart rendering is supported",
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
					&matchers.Is{
						Describe: "is rootFs with empty root path",
						Test: func(rootFs interface{}) bool {
							fs, ok := rootFs.(root.Fs)
							if !ok {
								return false
							}
							return fs.RootPath == "."

						},
					},
					&matchers.Is{
						Describe: "is github asset and has dest overridden",
						Test: func(asset interface{}) bool {
							githubAsset, ok := asset.(api.GitHubAsset)
							if !ok {
								return false
							}
							return strings.HasPrefix(githubAsset.Dest, ".ship/tmp/helmchart")

						},
					},
					[]libyaml.ConfigGroup{},
					"",
					api.ReleaseMetadata{},
					map[string]interface{}{},
				).Return(func(ctx context.Context) error { return nil })
			},
			expect:      "/helmchart",
			expectError: "",
		},
		{
			name:       "github fetches from github with '' root, event though rootFs has installer/",
			renderRoot: "installer/",
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
					&matchers.Is{
						Describe: "is rootFs with empty root path",
						Test: func(rootFs interface{}) bool {
							fs, ok := rootFs.(root.Fs)
							if !ok {
								return false
							}
							return fs.RootPath == "."

						},
					},
					&matchers.Is{
						Describe: "is github asset and has dest overridden",
						Test: func(asset interface{}) bool {
							githubAsset, ok := asset.(api.GitHubAsset)
							if !ok {
								return false
							}
							return strings.HasPrefix(githubAsset.Dest, ".ship/tmp/helmchart")
						},
					},
					[]libyaml.ConfigGroup{},
					"installer/",
					api.ReleaseMetadata{},
					map[string]interface{}{},
				).Return(func(ctx context.Context) error { return nil })
			},
			expect:      "/helmchart",
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
				test.asset,
				test.renderRoot,
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
				strings.Contains(dest, test.expect),
				"expected %s to have prefix %s",
				dest,
				test.expect,
			)
		})
	}
}
