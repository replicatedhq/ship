package kustomize

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/types"
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

func TestKustomizer_resolveExistingKustomize(t *testing.T) {
	tests := []struct {
		name       string
		original   []testFile
		InKust     *state.Kustomize
		overlayDir string
		WantKust   *state.Kustomize
		wantErr    bool
	}{
		{
			name: "no files in overlay dir, no current state",
			original: []testFile{
				{
					path:     "test/overlays/ship/placeholder",
					contents: "abc",
				},
			},
			overlayDir: "test/overlays/ship",
			InKust:     nil,
			WantKust:   nil,
		},
		{
			name: "no files in overlay dir, some current state",
			original: []testFile{
				{
					path:     "test/overlays/placeholder",
					contents: "abc",
				},
			},
			overlayDir: "test/overlays",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
		},
		{
			name: "garbled kustomization in current dir, some current state",
			original: []testFile{
				{
					path:     "test/overlays/kustomization.yaml",
					contents: "abc",
				},
			},
			overlayDir: "test/overlays",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
			wantErr: true,
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
		},
		{
			name: "overlays and resources files in overlay dir, some current state",
			original: []testFile{
				{
					path: "test/overlays/kustomization.yaml",
					contents: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../abc
resources:
- myresource.yaml
patchesStrategicMerge:
- mypatch.yaml
`,
				},
				{
					path:     "test/overlays/myresource.yaml",
					contents: `this is my resource`,
				},
				{
					path:     "test/overlays/mypatch.yaml",
					contents: `this is my patch`,
				},
			},
			overlayDir: "test/overlays",
			InKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"abc": "xyz"},
						Resources:     map[string]string{"abc": "xyz"},
						ExcludedBases: []string{"excludedBase"},
					},
				},
			},
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches:       map[string]string{"/mypatch.yaml": "this is my patch"},
						Resources:     map[string]string{"/myresource.yaml": "this is my resource"},
						ExcludedBases: []string{"excludedBase"},
						RawKustomize: types.Kustomization{
							TypeMeta:              types.TypeMeta{Kind: types.KustomizationKind, APIVersion: types.KustomizationVersion},
							PatchesStrategicMerge: []types.PatchStrategicMerge{"mypatch.yaml"},
							Resources:             []string{"myresource.yaml"},
							Bases:                 []string{"../abc"},
						},
					},
				},
			},
		},
		{
			name: "resources files in overlay dir, no current state",
			original: []testFile{
				{
					path: "test/overlays/kustomization.yaml",
					contents: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../abc
resources:
- myresource.yaml
- myotherresource.yaml
`,
				},
				{
					path:     "test/overlays/myresource.yaml",
					contents: `this is my resource`,
				},
				{
					path:     "test/overlays/myotherresource.yaml",
					contents: `this is my other resource`,
				},
			},
			overlayDir: "test/overlays",
			InKust:     nil,
			WantKust: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": state.Overlay{
						Patches: map[string]string{},
						Resources: map[string]string{
							"/myresource.yaml":      "this is my resource",
							"/myotherresource.yaml": "this is my other resource",
						},
						ExcludedBases: []string{},
						RawKustomize: types.Kustomization{
							TypeMeta: types.TypeMeta{Kind: types.KustomizationKind, APIVersion: types.KustomizationVersion},
							Resources: []string{
								"myresource.yaml",
								"myotherresource.yaml",
							},
							Bases: []string{"../abc"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			err := addTestFiles(fs, tt.original)
			req.NoError(err)

			manager, err := state.NewDisposableManager(log.NewNopLogger(), fs, viper.New())
			req.NoError(err)

			err = manager.SaveKustomize(tt.InKust)
			req.NoError(err)

			l := &Kustomizer{
				Logger: log.NewNopLogger(),
				FS:     fs,
				State:  manager,
			}

			err = l.resolveExistingKustomize(context.Background(), tt.overlayDir)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			outState, err := manager.CachedState()
			req.NoError(err)

			outKust := outState.CurrentKustomize()
			req.Equal(tt.WantKust, outKust)
		})
	}
}
