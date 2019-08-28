package util

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestShouldAddFileToBase(t *testing.T) {
	type file struct {
		contents string
		path     string
	}
	tests := []struct {
		name          string
		targetPath    string
		files         []file
		want          bool
		excludedPaths []string
	}{
		{name: "empty", targetPath: "", want: false, excludedPaths: []string{}},
		{name: "no extension", targetPath: "file", want: false, excludedPaths: []string{}},
		{name: "wrong extension", targetPath: "file.txt", want: false, excludedPaths: []string{}},
		{name: "yaml file", targetPath: "file.yaml", want: true, excludedPaths: []string{}},
		{name: "yml file", targetPath: "file.yml", want: true, excludedPaths: []string{}},
		{name: "kustomization yaml", targetPath: "kustomization.yaml", want: false, excludedPaths: []string{}},
		{name: "Chart yaml", targetPath: "Chart.yaml", want: false, excludedPaths: []string{}},
		{name: "values yaml", targetPath: "values.yaml", want: false, excludedPaths: []string{}},
		{name: "no extension in dir", targetPath: "dir/file", want: false, excludedPaths: []string{}},
		{name: "wrong extension in dir", targetPath: "dir/file.txt", want: false, excludedPaths: []string{}},
		{name: "yaml in dir", targetPath: "dir/file.yaml", want: true, excludedPaths: []string{}},
		{name: "yml in dir", targetPath: "dir/file.yml", want: true, excludedPaths: []string{}},
		{name: "kustomization yaml in dir", targetPath: "dir/kustomization.yaml", want: false, excludedPaths: []string{}},
		{name: "Chart yaml in dir", targetPath: "dir/Chart.yaml", want: false, excludedPaths: []string{}},
		{name: "values yaml in dir", targetPath: "dir/values.yaml", want: false, excludedPaths: []string{}},
		{name: "path in excluded", targetPath: "deployment.yaml", want: false, excludedPaths: []string{"/deployment.yaml"}},
		{name: "path not in excluded", targetPath: "service.yaml", want: true, excludedPaths: []string{"/deployment.yaml"}},
		{name: "similar path in excluded", targetPath: "dir/service.yaml", want: true, excludedPaths: []string{"/service.yaml"}},
		{name: "non-k8s yaml file", targetPath: "file.yaml", want: false, excludedPaths: []string{}, files: []file{{contents: "a: b", path: "file.yaml"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, file := range tt.files {
				req.NoError(fs.WriteFile(file.path, []byte(file.contents), os.FileMode(777)))
			}

			got := ShouldAddFileToBase(&fs, tt.excludedPaths, tt.targetPath)

			req.Equal(tt.want, got, "expected %t for path %s, got %t", tt.want, tt.targetPath, got)
		})
	}
}

func TestIsK8sYaml(t *testing.T) {
	type file struct {
		contents string
		path     string
	}
	tests := []struct {
		name   string
		files  []file
		target string
		want   bool
	}{
		{
			name:   "file does not exist",
			files:  []file{},
			target: "myfile.yaml",
			want:   true,
		},
		{
			name: "invalid file",
			files: []file{
				{
					path: "dir/myfile.yaml",
					contents: `
this is not valid k8s yaml
`,
				},
			},
			target: "dir/myfile.yaml",
			want:   false,
		},
		{
			name: "file does not exist but another does",
			files: []file{
				{
					path: "notmyfile.yaml",
					contents: `
kind: Something
metadata:
  name: something
`,
				},
			},
			target: "myfile.yaml",
			want:   true,
		},
		{
			name: "valid file in subdir",
			files: []file{
				{
					path: "dir/myfile.yaml",
					contents: `
kind: Something
metadata:
  name: something
`,
				},
			},
			target: "dir/myfile.yaml",
			want:   true,
		},
		{
			name: "missing name, not a list",
			files: []file{
				{
					path: "dir/notlist.yaml",
					contents: `
kind: notalisttype
metadata:
  namespace: notaname
`,
				},
			},
			target: "dir/notlist.yaml",
			want:   false,
		},
		{
			name: "missing name, is a list",
			files: []file{
				{
					path: "dir/islist.yaml",
					contents: `
kind: aList
`,
				},
			},
			target: "dir/islist.yaml",
			want:   true,
		},
		{
			name: "missing kind",
			files: []file{
				{
					path: "dir/nokind.yaml",
					contents: `
notkind: aList
`,
				},
			},
			target: "dir/nokind.yaml",
			want:   false,
		},
		{
			name: "multidoc yaml",
			files: []file{
				{
					path: "dir/multidoc.yaml",
					contents: `
---
# Source: concourse/templates/web-rolebinding.yaml

---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: concourse-web-main
  namespace: concourse-main
  labels:
    app: concourse-web
    chart: concourse-3.7.2
    heritage: Tiller
    release: concourse
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: concourse-web
subjects:
- kind: ServiceAccount
  name: concourse-web
  namespace: default
`,
				},
			},
			target: "dir/multidoc.yaml",
			want:   true,
		},
		{
			name: "multidoc yaml, both unacceptable",
			files: []file{
				{
					path: "dir/multidoc.yaml",
					contents: `
---
# Source: concourse/templates/web-rolebinding.yaml

---
metadata:
  name: concourse-web-main
`,
				},
			},
			target: "dir/multidoc.yaml",
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, file := range tt.files {
				req.NoError(fs.WriteFile(file.path, []byte(file.contents), os.FileMode(777)))
			}

			got := IsK8sYaml(&fs, tt.target)
			req.Equal(tt.want, got, "IsK8sYaml() = %v, want %v for file %s", got, tt.want, tt.target)
		})
	}
}
