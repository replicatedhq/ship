package kustomize

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
)

type testFile struct {
	path     string
	contents string
}

func addTestFiles(fs afero.Afero, testFiles []testFile) error {
	for _, testFile := range testFiles {
		if err := fs.MkdirAll(filepath.Dir(testFile.path), 0755); err != nil {
			return err
		}
		if err := fs.WriteFile(testFile.path, []byte(testFile.contents), 0644); err != nil {
			return err
		}
	}
	return nil
}

func readTestFiles(step api.Kustomize, fs afero.Afero) ([]testFile, error) {
	files := []testFile{}
	if err := fs.Walk(step.Base, func(targetPath string, info os.FileInfo, err error) error {
		if filepath.Ext(targetPath) == ".yaml" {
			contents, err := fs.ReadFile(targetPath)
			if err != nil {
				return err
			}

			files = append(files, testFile{
				path:     targetPath,
				contents: string(contents),
			})
		}
		return nil
	}); err != nil {
		return files, err
	}

	return files, nil
}

func TestKustomizer_replaceOriginal(t *testing.T) {

	tests := []struct {
		name     string
		step     api.Kustomize
		built    []util.PostKustomizeFile
		original []testFile
		expect   []testFile
		wantErr  bool
	}{
		{
			name: "replace single file",
			step: api.Kustomize{
				Base: "",
			},
			built: []util.PostKustomizeFile{
				{
					Minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "strawberry",
						},
					},
					Full: map[string]interface{}{
						"kind": "Fruit",
						"metadata": map[string]interface{}{
							"name": "strawberry",
						},
						"spec": map[string]interface{}{
							"modified": "modified",
						},
					},
				},
			},
			original: []testFile{
				{
					path: "strawberry.yaml",
					contents: `kind: Fruit
metadata:
  name: strawberry
spec:
  original: original
`,
				},
			},
			expect: []testFile{
				{
					path: "strawberry.yaml",
					contents: `kind: Fruit
metadata:
  name: strawberry
spec:
  modified: modified
`,
				},
			},
		},
		{
			name: "skip CRDs",
			step: api.Kustomize{
				Base: "",
			},
			built: []util.PostKustomizeFile{
				{
					Minimal: util.MinimalK8sYaml{
						Kind: "CustomResourceDefinition",
						Metadata: util.MinimalK8sMetadata{
							Name: "strawberry",
						},
					},
					Full: map[string]interface{}{
						"kind": "CustomResourceDefinition",
						"metadata": map[string]interface{}{
							"name": "strawberry",
						},
						"spec": map[string]interface{}{
							"modified": "modified",
						},
					},
				},
			},
			original: []testFile{
				{
					path: "strawberry.yaml",
					contents: `kind: CustomResourceDefinition
metadata:
  name: strawberry
spec:
  original: original
`,
				},
			},
			expect: []testFile{
				{
					path: "strawberry.yaml",
					contents: `kind: CustomResourceDefinition
metadata:
  name: strawberry
spec:
  original: original
`,
				},
			},
		},
		{
			name: "replace nested file",
			step: api.Kustomize{
				Base: "",
			},
			built: []util.PostKustomizeFile{
				{
					Minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "banana",
						},
					},
					Full: map[string]interface{}{
						"kind": "Fruit",
						"metadata": map[string]interface{}{
							"name": "banana",
						},
						"spec": map[string]interface{}{
							"modified": "modified",
						},
					},
				},
			},
			original: []testFile{
				{
					path: "somedir/banana.yaml",
					contents: `kind: Fruit
metadata:
  name: banana
spec:
  original: original
`,
				},
			},
			expect: []testFile{
				{
					path: "somedir/banana.yaml",
					contents: `kind: Fruit
metadata:
  name: banana
spec:
  modified: modified
`,
				},
			},
		},
		{
			name: "replace multiple files",
			step: api.Kustomize{
				Base: "",
			},
			built: []util.PostKustomizeFile{
				{
					Minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "dragonfruit",
						},
					},
					Full: map[string]interface{}{
						"kind": "Fruit",
						"metadata": map[string]interface{}{
							"name": "dragonfruit",
						},
						"spec": map[string]interface{}{
							"modified": "modified dragonfruit",
						},
					},
				},
				{
					Minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "pomegranate",
						},
					},
					Full: map[string]interface{}{
						"kind": "Fruit",
						"metadata": map[string]interface{}{
							"name": "pomegranate",
						},
						"spec": map[string]interface{}{
							"modified": "modified pomegranate",
						},
					},
				},
			},
			original: []testFile{
				{
					path: "somedir/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "pomegranate.yaml",
					contents: `kind: Fruit
metadata:
  name: pomegranate
spec:
  original: original pomegranate
`,
				},
			},
			expect: []testFile{
				{
					path: "somedir/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  modified: modified dragonfruit
`,
				},
				{
					path: "pomegranate.yaml",
					contents: `kind: Fruit
metadata:
  name: pomegranate
spec:
  modified: modified pomegranate
`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			err := addTestFiles(mockFs, tt.original)
			req.NoError(err)

			l := &Kustomizer{
				Logger: log.NewNopLogger(),
				FS:     mockFs,
			}

			err = l.replaceOriginal(tt.step.Base, tt.built)
			req.NoError(err)

			actual, err := readTestFiles(tt.step, mockFs)
			req.NoError(err)

			req.ElementsMatch(tt.expect, actual)
		})
	}
}

func TestKustomizer_containsBase(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		original []testFile
		want     string
		wantErr  bool
	}{
		{
			name: "dir does not exist",
			path: "abc",
			original: []testFile{
				{
					path: "xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "other yaml in dir",
			path: "abc",
			original: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "irrelevant kustomization yaml in dir",
			path: "abc",
			original: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "abc/kustomization.yaml",
					contents: `
kind: ""
apiversion: ""
resources:
- dragonfruit.yaml
`,
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "relevant kustomization yaml in dir",
			path: "abc",
			original: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "abc/kustomization.yaml",
					contents: `
kind: ""
apiversion: ""
resources:
- dragonfruit.yaml
bases:
- ../base
`,
				},
			},
			want:    "base",
			wantErr: false,
		},
		{
			name: "unparseable kustomization yaml in dir",
			path: "abc",
			original: []testFile{
				{
					path: "abc/kustomization.yaml",
					contents: `
thisisnotvalidyaml
`,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "multiple base kustomization yaml in dir",
			path: "abc",
			original: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "abc/kustomization.yaml",
					contents: `
kind: ""
apiversion: ""
resources:
- dragonfruit.yaml
bases:
- ../base
- ../otherbase
`,
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			err := addTestFiles(mockFs, tt.original)
			req.NoError(err)

			l := &Kustomizer{
				Logger: log.NewNopLogger(),
				FS:     mockFs,
			}
			got, err := l.containsBase(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Kustomizer.containsBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Kustomizer.containsBase() = %v, want %v", got, tt.want)
			}
		})
	}
}
