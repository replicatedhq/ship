package github

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func Test_getDestPath(t *testing.T) {
	type args struct {
		githubPath string
		asset      api.GitHubAsset
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "basic file",
			args: args{
				githubPath: "README.md",
				asset: api.GitHubAsset{
					Path:      "README.md",
					StripPath: "",
					AssetShared: api.AssetShared{
						Dest: "./",
					},
				},
			},
			want:    "README.md",
			wantErr: false,
		},
		{
			name: "file in subdir",
			args: args{
				githubPath: "subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "subdir/",
					StripPath: "",
					AssetShared: api.AssetShared{
						Dest: "./",
					},
				},
			},
			want:    "subdir/README.md",
			wantErr: false,
		},
		{
			name: "file in subdir with dest dir",
			args: args{
				githubPath: "subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "subdir/",
					StripPath: "",
					AssetShared: api.AssetShared{
						Dest: "./dest",
					},
				},
			},
			want:    "dest/subdir/README.md",
			wantErr: false,
		},
		{
			name: "file in stripped subdir with dest dir",
			args: args{
				githubPath: "subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "subdir/",
					StripPath: "true",
					AssetShared: api.AssetShared{
						Dest: "./dest",
					},
				},
			},
			want:    "dest/README.md",
			wantErr: false,
		},
		{
			name: "literal file in stripped subdir with dest dir",
			args: args{
				githubPath: "dir/subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "dir/subdir/README.md",
					StripPath: "true",
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "dest/README.md",
			wantErr: false,
		},
		{
			name: "file in stripped subdir that lacks a trailing slash with dest dir",
			args: args{
				githubPath: "dir/subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "dir/subdir",
					StripPath: "true",
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "dest/README.md",
			wantErr: false,
		},
		{
			name: "templated dest dir",
			args: args{
				githubPath: "dir/subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "dir/subdir",
					StripPath: "false",
					AssetShared: api.AssetShared{
						Dest: "dest{{repl Add 1 1}}",
					},
				},
			},
			want:    "dest2/dir/subdir/README.md",
			wantErr: false,
		},
		{
			name: "templated stripPath (eval to true)",
			args: args{
				githubPath: "dir/subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "dir/subdir",
					StripPath: `{{repl ParseBool "true"}}`,
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "dest/README.md",
			wantErr: false,
		},
		{
			name: "templated stripPath (eval to false)",
			args: args{
				githubPath: "dir/subdir/README.md",
				asset: api.GitHubAsset{
					Path:      "dir/subdir",
					StripPath: `{{repl ParseBool "false"}}`,
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "dest/dir/subdir/README.md",
			wantErr: false,
		},
		{
			name: "strip path of root dir file",
			args: args{
				githubPath: "README.md",
				asset: api.GitHubAsset{
					Path:      "",
					StripPath: "true",
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "dest/README.md",
			wantErr: false,
		},
		{
			name: "not a valid template function (dest)",
			args: args{
				githubPath: "README.md",
				asset: api.GitHubAsset{
					Path:      "",
					StripPath: "true",
					AssetShared: api.AssetShared{
						Dest: "{{repl NotATemplateFunction }}",
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "not a valid template function (stripPath)",
			args: args{
				githubPath: "README.md",
				asset: api.GitHubAsset{
					Path:      "",
					StripPath: "{{repl NotATemplateFunction }}",
					AssetShared: api.AssetShared{
						Dest: "dest",
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			bb := templates.NewBuilderBuilder(testLogger, v)
			builder, err := bb.FullBuilder(api.ReleaseMetadata{}, []libyaml.ConfigGroup{}, map[string]interface{}{})
			req.NoError(err)

			got, err := getDestPath(tt.args.githubPath, tt.args.asset, builder)
			if !tt.wantErr {
				req.NoErrorf(err, "getDestPath(%s, %+v, builder) error = %v", tt.args.githubPath, tt.args.asset, err)
			} else {
				req.Error(err)
			}

			// convert the returned file to forwardslash format before testing - otherwise this test fails when the separator isn't '/'
			req.Equal(tt.want, filepath.ToSlash(got))
		})
	}
}

func Test_getDestPathNoProxy(t *testing.T) {
	tests := []struct {
		name    string
		asset   api.GitHubAsset
		want    string
		wantErr bool
	}{
		{
			name: "basic file",
			asset: api.GitHubAsset{
				Path:      "README.md",
				StripPath: "",
				AssetShared: api.AssetShared{
					Dest: "./",
				},
			},
			want:    "installer/README.md",
			wantErr: false,
		},
		{
			name: "file in subdir",
			asset: api.GitHubAsset{
				Path:      "subdir/README.md",
				StripPath: "",
				AssetShared: api.AssetShared{
					Dest: "./",
				},
			},
			want:    "installer/subdir/README.md",
			wantErr: false,
		},
		{
			name: "file in subdir with dest dir",
			asset: api.GitHubAsset{
				Path:      "subdir/README.md",
				StripPath: "",
				AssetShared: api.AssetShared{
					Dest: "./dest",
				},
			},
			want:    "installer/dest/subdir/README.md",
			wantErr: false,
		},
		{
			name: "file in stripped subdir with dest dir",
			asset: api.GitHubAsset{
				Path:      "subdir/README.md",
				StripPath: "true",
				AssetShared: api.AssetShared{
					Dest: "./dest",
				},
			},
			want:    "installer/dest/README.md",
			wantErr: false,
		},
		{
			name: "literal file in stripped subdir with dest dir",
			asset: api.GitHubAsset{
				Path:      "dir/subdir/README.md",
				StripPath: "true",
				AssetShared: api.AssetShared{
					Dest: "dest",
				},
			},
			want:    "installer/dest/README.md",
			wantErr: false,
		},
		{
			name: "templated dest dir",
			asset: api.GitHubAsset{
				Path:      "dir/subdir/README.md",
				StripPath: "false",
				AssetShared: api.AssetShared{
					Dest: "dest{{repl Add 1 1}}",
				},
			},
			want:    "installer/dest2/dir/subdir/README.md",
			wantErr: false,
		},
		{
			name: "templated stripPath (eval to true)",
			asset: api.GitHubAsset{
				Path:      "dir/subdir/README.md",
				StripPath: `{{repl ParseBool "true"}}`,
				AssetShared: api.AssetShared{
					Dest: "dest",
				},
			},
			want:    "installer/dest/README.md",
			wantErr: false,
		},
		{
			name: "templated stripPath (eval to false)",
			asset: api.GitHubAsset{
				Path:      "dir/subdir/README.md",
				StripPath: `{{repl ParseBool "false"}}`,
				AssetShared: api.AssetShared{
					Dest: "dest",
				},
			},
			want:    "installer/dest/dir/subdir/README.md",
			wantErr: false,
		},
		{
			name: "strip path of root dir file",
			asset: api.GitHubAsset{
				Path:      "README.md",
				StripPath: "true",
				AssetShared: api.AssetShared{
					Dest: "dest",
				},
			},
			want:    "installer/dest/README.md",
			wantErr: false,
		},
		{
			name: "not a valid template function (dest)",
			asset: api.GitHubAsset{
				Path:      "README.md",
				StripPath: "true",
				AssetShared: api.AssetShared{
					Dest: "{{repl NotATemplateFunction }}",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "not a valid template function (stripPath)",
			asset: api.GitHubAsset{
				Path:      "README.md",
				StripPath: "{{repl NotATemplateFunction }}",
				AssetShared: api.AssetShared{
					Dest: "dest",
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			bb := templates.NewBuilderBuilder(testLogger, v)
			builder, err := bb.FullBuilder(api.ReleaseMetadata{}, []libyaml.ConfigGroup{}, map[string]interface{}{})
			req.NoError(err)

			got, err := getDestPathNoProxy(tt.asset, builder)
			if !tt.wantErr {
				req.NoError(err)
			} else {
				req.Error(err)
			}

			// convert the returned file to forwardslash format before testing - otherwise this test fails when the separator isn't '/'
			req.Equal(tt.want, filepath.ToSlash(got))
		})
	}
}

func Test_filterGithubContents(t *testing.T) {
	type args struct {
		githubContents []api.GithubContent
		asset          api.GitHubAsset
	}
	tests := []struct {
		name string
		args args
		want []api.GithubFile
	}{
		{
			name: "has slash prefix and suffix",
			args: args{
				githubContents: []api.GithubContent{{
					Path:  "subdir",
					Files: []api.GithubFile{{Name: "1"}},
				}},
				asset: api.GitHubAsset{
					Path: "/subdir/",
				},
			},
			want: []api.GithubFile{{Name: "1"}},
		},
		{
			name: "is root",
			args: args{
				githubContents: []api.GithubContent{{
					Path:  "/",
					Files: []api.GithubFile{{Name: "1"}},
				}},
				asset: api.GitHubAsset{
					Path: "/",
				},
			},
			want: []api.GithubFile{{Name: "1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterGithubContents(tt.args.githubContents, tt.args.asset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterGithubContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
