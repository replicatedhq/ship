package kustomize

import (
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/replicatedhq/ship/pkg/constants"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	kustomizepatch "sigs.k8s.io/kustomize/pkg/patch"
	k8stypes "sigs.k8s.io/kustomize/pkg/types"
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
				{
					path: "strawberry/service.yaml",
					contents: `apiVersion: v1
kind: Service
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
				Bases: []string{"../../strawberry"},
				PatchesJson6902: []kustomizepatch.PatchJson6902{
					{
						Path: "tiller-patch.json",
						Target: &kustomizepatch.Target{
							Group:   "apps",
							Kind:    "Deployment",
							Name:    "strawberry",
							Version: "v1beta2",
						},
					},
					{
						Path: "tiller-patch.json",
						Target: &kustomizepatch.Target{
							Kind:    "Service",
							Name:    "strawberry",
							Version: "v1",
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
				Bases: []string{"../../banana"},
				PatchesJson6902: []kustomizepatch.PatchJson6902{
					{
						Path: "tiller-patch.json",
						Target: &kustomizepatch.Target{
							Group:   "apps",
							Kind:    "Deployment",
							Name:    "banana",
							Version: "v1beta2",
						},
					},
					{
						Path: "tiller-patch.json",
						Target: &kustomizepatch.Target{
							Kind:    "Service",
							Name:    "banana",
							Version: "v1",
						},
					},
				},
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

			l := &Kustomizer{
				Logger: log.NewNopLogger(),
				FS:     mockFs,
			}
			err := l.generateTillerPatches(tt.step)
			req.NoError(err)

			kustomizationB, err := mockFs.ReadFile(path.Join(constants.TempApplyOverlayPath, "kustomization.yaml"))
			req.NoError(err)

			kustomizationYaml := k8stypes.Kustomization{}
			err = yaml.Unmarshal(kustomizationB, &kustomizationYaml)
			req.NoError(err)

			req.Equal(tt.expectKustomization, kustomizationYaml)
		})
	}
}
