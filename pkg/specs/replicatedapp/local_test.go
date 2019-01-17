package replicatedapp

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestLoadLocalGitHubContents(t *testing.T) {
	tests := []struct {
		name           string
		githubContent  []string
		fs             map[string]string
		expectContents []GithubContent
	}{
		{
			name:           "none to set",
			githubContent:  nil,
			expectContents: nil,
		},
		{
			name: "set one file",
			fs: map[string]string{
				"/foo/bar.txt": "some-contents",
			},
			githubContent: []string{"replicatedhq/test-stuff:/bar.txt:master:/foo"},
			expectContents: []GithubContent{
				{
					Repo: "replicatedhq/test-stuff",
					Path: "/bar.txt",
					Ref:  "master",
					Files: []GithubFile{
						{
							Path: "/bar.txt",
							Name: "bar.txt",
							Size: 13,
							Sha:  "6e32ea34db1b3755d7dec972eb72c705338f0dd8e0be881d966963438fb2e800",
							Data: "c29tZS1jb250ZW50cw==",
						},
					},
				},
			},
		},
		{
			name: "set many files from two repos",
			fs: map[string]string{
				"/foo/bar.txt":     "some-contents",
				"/foo/baz.txt":     "some-contents",
				"/foo/bar/baz.txt": "some-contents",
				"/spam/eggs.txt":   "some-other-contents",
			},
			githubContent: []string{
				"replicatedhq/test-stuff:/:master:/foo",
				"replicatedhq/other-tests:/eggs.txt:release:/spam",
			},
			expectContents: []GithubContent{
				{
					Repo: "replicatedhq/test-stuff",
					Path: "/",
					Ref:  "master",
					Files: []GithubFile{
						{
							Path: "/bar/baz.txt",
							Name: "baz.txt",
							Size: 13,
							Sha:  "6e32ea34db1b3755d7dec972eb72c705338f0dd8e0be881d966963438fb2e800",
							Data: "c29tZS1jb250ZW50cw==",
						},
						{
							Path: "/bar.txt",
							Name: "bar.txt",
							Size: 13,
							Sha:  "6e32ea34db1b3755d7dec972eb72c705338f0dd8e0be881d966963438fb2e800",
							Data: "c29tZS1jb250ZW50cw==",
						},
						{
							Path: "/baz.txt",
							Name: "baz.txt",
							Size: 13,
							Sha:  "6e32ea34db1b3755d7dec972eb72c705338f0dd8e0be881d966963438fb2e800",
							Data: "c29tZS1jb250ZW50cw==",
						},
					},
				},
				{
					Repo: "replicatedhq/other-tests",
					Path: "/eggs.txt",
					Ref:  "release",
					Files: []GithubFile{
						{
							Path: "/eggs.txt",
							Name: "eggs.txt",
							Size: 19,
							Sha:  "a2c0a8c54d71e14e9533749c32716c12f92f61294dfdce4f3b4c07303c0119b0",
							Data: "c29tZS1vdGhlci1jb250ZW50cw==",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}

			for key, value := range test.fs {
				err := mockFs.WriteFile(key, []byte(value), 0777)
				req.NoError(err)
			}

			resolver := &resolver{
				Logger:            &logger.TestLogger{T: t},
				FS:                mockFs,
				SetGitHubContents: test.githubContent,
			}

			result, err := resolver.loadLocalGitHubContents()

			req.NoError(err)
			req.Equal(test.expectContents, result)
		})
	}
}
