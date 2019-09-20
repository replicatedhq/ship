package kustomize

import (
	"path"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	k8stypes "sigs.k8s.io/kustomize/v3/pkg/types"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
)

func TestKustomizer_generateTillerPatches(t *testing.T) {
	type testFile struct {
		path     string
		contents string
	}
	tests := []struct {
		name                string
		step                api.Kustomize
		testFiles           []testFile
		expectKustomization k8stypes.Kustomization
	}{
		{
			name: "yaml with heritage and chart labels",
			step: api.Kustomize{
				Base: "strawberry",
			},
			testFiles: []testFile{
				{
					path: "strawberry/deployment.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:    []string{"../../strawberry"},
				PatchesJson6902: []k8stypes.PatchJson6902{
					{
						Path: "chart-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "strawberry",
						},
					},
					{
						Path: "heritage-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "strawberry",
						},
					},
				},
			},
		},
		{
			name: "kustomization yaml",
			step: api.Kustomize{
				Base: "strawberry",
			},
			testFiles: []testFile{
				{
					path: "strawberry/kustomization.yaml",
					contents: `apiVersion: apps/v1beta2
bases:
- ../../base
patchesJson6902:
- path: chart-patch.json
  target:
    group: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cert-manager-cainjector
    version: v1beta1
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta:        k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:           []string{"../../strawberry"},
				PatchesJson6902: nil,
			},
		},
		{
			name: "both kustomization and relevant yaml with heritage and chart labels",
			step: api.Kustomize{
				Base: "strawberry",
			},
			testFiles: []testFile{
				{
					path: "strawberry/deployment.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry
`,
				},
				{
					path: "strawberry/kustomization.yaml",
					contents: `apiVersion: apps/v1beta2
bases:
- ../../base
patchesJson6902:
- path: chart-patch.json
  target:
    group: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cert-manager-cainjector
    version: v1beta1
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:    []string{"../../strawberry"},
				PatchesJson6902: []k8stypes.PatchJson6902{
					{
						Path: "chart-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "strawberry",
						},
					},
					{
						Path: "heritage-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "strawberry",
						},
					},
				},
			},
		},
		{
			name: "yaml with only heritage label",
			step: api.Kustomize{
				Base: "pomegranate",
			},
			testFiles: []testFile{
				{
					path: "pomegranate/deployment.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: pomegranate
    heritage: Tiller
  name: pomegranate
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:    []string{"../../pomegranate"},
				PatchesJson6902: []k8stypes.PatchJson6902{
					{
						Path: "heritage-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "pomegranate",
						},
					},
				},
			},
		},
		{
			name: "yaml with only chart label",
			step: api.Kustomize{
				Base: "apple",
			},
			testFiles: []testFile{
				{
					path: "apple/deployment.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: apple
    chart: apple-1.0.0
  name: apple
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:    []string{"../../apple"},
				PatchesJson6902: []k8stypes.PatchJson6902{
					{
						Path: "chart-patch.json",
						Target: &k8stypes.PatchTarget{
							Gvk:  gvk.Gvk{Group: "apps", Kind: "Deployment", Version: "v1beta2"},
							Name: "apple",
						},
					},
				},
			},
		},
		{
			name: "yaml without heritage and chart labels",
			step: api.Kustomize{
				Base: "banana",
			},
			testFiles: []testFile{
				{
					path: "banana/deployment.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: banana
  name: banana
`,
				},
				{
					path: "banana/service.yaml",
					contents: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: banana
  name: banana
`,
				},
			},
			expectKustomization: k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
				Bases:    []string{"../../banana"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, testFile := range tt.testFiles {
				err := mockFs.WriteFile(testFile.path, []byte(testFile.contents), 0755)
				req.NoError(err)
			}

			stateManager, err := state.NewDisposableManager(log.NewNopLogger(), mockFs, viper.New())
			req.NoError(err)

			err = stateManager.Save(state.State{V1: &state.V1{Kustomize: &state.Kustomize{}}})
			req.NoError(err)

			l := &Kustomizer{
				Logger: log.NewNopLogger(),
				FS:     mockFs,
				State:  stateManager,
			}
			err = l.generateTillerPatches(tt.step)
			req.NoError(err)

			kustomizationB, err := mockFs.ReadFile(path.Join(constants.DefaultOverlaysPath, "kustomization.yaml"))
			req.NoError(err)

			kustomizationYaml := k8stypes.Kustomization{
				TypeMeta: k8stypes.TypeMeta{Kind: k8stypes.KustomizationKind, APIVersion: k8stypes.KustomizationVersion},
			}
			err = yaml.Unmarshal(kustomizationB, &kustomizationYaml)
			req.NoError(err)

			req.Equal(tt.expectKustomization, kustomizationYaml)
		})
	}
}
