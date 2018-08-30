package util

import "testing"

func TestIsGithubURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "empty",
			url:  "",
			want: false,
		},
		{
			name: "random nonsense",
			url:  "a string of random nonsense",
			want: false,
		},
		{
			name: "mocked github url with tree",
			url:  "github.com/OWNER/REPO/tree/REF/SUBDIR",
			want: true,
		},
		{
			name: "ship repo in pkg dir on master",
			url:  "https://github.com/replicatedhq/ship/tree/master/pkg",
			want: true,
		},
		{
			name: "ship repo in pkg/specs dir on master",
			url:  "https://github.com/replicatedhq/ship/tree/master/pkg/specs",
			want: true,
		},
		{
			name: "ship repo in pkg/specs dir at hash with www",
			url:  "https://www.github.com/replicatedhq/ship/tree/atestsha/pkg/specs",
			want: true,
		},
		{
			name: "ship repo in root dir on master with www",
			url:  "https://www.github.com/replicatedhq/ship/tree/master",
			want: true,
		},
		{
			name: "github repo with no tree",
			url:  "github.com/replicatedhq/ship",
			want: true,
		},
		{
			name: "github repo with no tree with www",
			url:  "https://www.github.com/replicatedhq/ship",
			want: true,
		},
		{
			name: "github repo with no tree with subdir",
			url:  "https://github.com/replicatedhq/ship/pkg/specs",
			want: true,
		},
		{
			name: "github repo with no https or tree with subdir",
			url:  "github.com/replicatedhq/ship/pkg/specs",
			want: true,
		},
		{
			name: "bitbucket repo",
			url:  "bitbucket.org/ww/goautoneg",
			want: false,
		},
		{
			name: "already configured go-getter string",
			url:  "github.com/replicatedhq/ship?ref=master//pkg/specs",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGithubURL(tt.url); got != tt.want {
				t.Errorf("IsGithubURL(%s) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParseGithubURL(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    GithubURL
		wanterr bool
	}{
		{
			name:    "empty",
			path:    "",
			wanterr: true,
		},
		{
			name:    "helm chart",
			path:    "stable/mysql",
			wanterr: true,
		},
		{
			name:    "random nonsense",
			path:    "a string of random nonsense",
			wanterr: true,
		},
		{
			name:    "bitbucket repo",
			path:    "bitbucket.org/ww/goautoneg",
			wanterr: true,
		},
		{
			name:    "already configured go-getter string",
			path:    "github.com/replicatedhq/ship?ref=master//pkg/specs",
			wanterr: true,
		},
		{
			name: "mocked github url with tree",
			path: "github.com/OWNER/REPO/tree/REF/SUBDIR",
			want: GithubURL{
				Owner:  "OWNER",
				Repo:   "REPO",
				Ref:    "REF",
				Subdir: "SUBDIR",
			},
		},
		{
			name: "ship repo in pkg dir on master",
			path: "https://github.com/replicatedhq/ship/tree/master/pkg",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "master",
				Subdir: "pkg",
			},
		},
		{
			name: "ship repo in pkg/specs dir on master",
			path: "https://github.com/replicatedhq/ship/tree/master/pkg/specs",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "master",
				Subdir: "pkg/specs",
			},
		},
		{
			name: "ship repo in pkg/specs dir at hash with www",
			path: "https://www.github.com/replicatedhq/ship/tree/atestsha/pkg/specs",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "atestsha",
				Subdir: "pkg/specs",
			},
		},
		{
			name: "ship repo in root dir on master with www",
			path: "https://www.github.com/replicatedhq/ship/tree/master",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "master",
				Subdir: "",
			},
		},
		{
			name: "github repo with no tree",
			path: "https://github.com/replicatedhq/ship",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "default",
				Subdir: "",
			},
		},
		{
			name: "github repo with no tree with www",
			path: "https://www.github.com/replicatedhq/ship",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "default",
				Subdir: "",
			},
		},
		{
			name: "github repo with no tree with subdir",
			path: "https://github.com/replicatedhq/ship/pkg/specs",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "default",
				Subdir: "pkg/specs",
			},
		},
		{
			name: "github repo with no https or tree with subdir",
			path: "github.com/replicatedhq/ship/pkg/specs",
			want: GithubURL{
				Owner:  "replicatedhq",
				Repo:   "ship",
				Ref:    "default",
				Subdir: "pkg/specs",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGithubURL(tt.path, "default")
			if err != nil {
				if tt.wanterr {
					return
				} else {
					t.Errorf("got unexpected error %s parsing %q", err.Error(), tt.path)
				}
			}

			if got != tt.want {
				t.Errorf("untreeGithub(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
