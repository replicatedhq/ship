package gogetter

import (
	"testing"
)

func TestIsGoGettable(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "empty",
			path: "",
			want: false,
		},
		{
			name: "helm chart",
			path: "stable/mysql",
			want: false,
		},
		{
			name: "github",
			path: "github.com/replicatedhq/ship",
			want: true,
		},
		{
			name: "bitbucket",
			// this needs to be a valid bitbucket repo to work
			path: "bitbucket.org/ww/goautoneg",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGoGettable(tt.path); got != tt.want {
				t.Errorf("isGoGettable(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestUntreeGithub(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty",
			path: "",
			want: "",
		},
		{
			name: "helm chart",
			path: "stable/mysql",
			want: "stable/mysql",
		},
		{
			name: "random nonsense",
			path: "a string of random nonsense",
			want: "a string of random nonsense",
		},
		{
			name: "bitbucket repo",
			path: "bitbucket.org/ww/goautoneg",
			want: "bitbucket.org/ww/goautoneg",
		},
		{
			name: "already configured go-getter string",
			path: "github.com/replicatedhq/ship?ref=master//pkg/specs",
			want: "github.com/replicatedhq/ship?ref=master//pkg/specs",
		},
		{
			name: "mocked github url with tree",
			path: "github.com/OWNER/REPO/tree/REF/SUBDIR",
			want: "github.com/OWNER/REPO?ref=REF//SUBDIR",
		},
		{
			name: "ship repo in pkg dir on master",
			path: "https://github.com/replicatedhq/ship/tree/master/pkg",
			want: "github.com/replicatedhq/ship?ref=master//pkg",
		},
		{
			name: "ship repo in pkg/specs dir on master",
			path: "https://github.com/replicatedhq/ship/tree/master/pkg/specs",
			want: "github.com/replicatedhq/ship?ref=master//pkg/specs",
		},
		{
			name: "ship repo in pkg/specs dir at hash with www",
			path: "https://www.github.com/replicatedhq/ship/tree/atestsha/pkg/specs",
			want: "github.com/replicatedhq/ship?ref=atestsha//pkg/specs",
		},
		{
			name: "ship repo in root dir on master with www",
			path: "https://www.github.com/replicatedhq/ship/tree/master",
			want: "github.com/replicatedhq/ship?ref=master//",
		},
		{
			name: "github repo with no tree",
			path: "https://github.com/replicatedhq/ship",
			want: "github.com/replicatedhq/ship?ref=master//",
		},
		{
			name: "github repo with no tree with www",
			path: "https://www.github.com/replicatedhq/ship",
			want: "github.com/replicatedhq/ship?ref=master//",
		},
		{
			name: "github repo with no tree with subdir",
			path: "https://github.com/replicatedhq/ship/pkg/specs",
			want: "github.com/replicatedhq/ship?ref=master//pkg/specs",
		},
		{
			name: "github repo with no https or tree with subdir",
			path: "github.com/replicatedhq/ship/pkg/specs",
			want: "github.com/replicatedhq/ship?ref=master//pkg/specs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UntreeGithub(tt.path); got != tt.want {
				t.Errorf("untreeGithub(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
