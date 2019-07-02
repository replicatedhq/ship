package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/pkg/gvk"
	"sigs.k8s.io/kustomize/pkg/resid"
)

func TestExcludeKubernetesResource(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name             string
		basePath         string
		excludedResource string
		wantErr          bool
		inputFiles       []fileStruct
		outputFiles      []fileStruct
		wantResIDs       []resid.ResId
	}{
		{
			name:             "existsInBase",
			basePath:         "base",
			excludedResource: "/myresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
		{
			name:             "does not exist",
			basePath:         "base",
			excludedResource: "notexist-resource.yaml",
			wantErr:          true,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
		{
			name:             "exists in child base",
			basePath:         "base",
			excludedResource: "/another-base/anotherresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
bases:
- ../another/base
resources:
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- anotherresource.yaml
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
bases:
- ../another/base
resources:
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			wantResIDs: []resid.ResId{resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector")},
		},
		{
			name:             "already removed",
			basePath:         "base",
			excludedResource: "myresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.MkdirAll(filepath.Dir(inFile.name), os.FileMode(0644)))
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			// run exclude function
			actualResIDs, err := ExcludeKubernetesResource(mockFs, tt.basePath, tt.excludedResource)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExcludeKubernetesResource() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.outputFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			err = mockFs.Walk("", func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					actualFileNames = append(actualFileNames, path)
				}
				return nil
			})
			req.NoError(err, "read output /")

			req.ElementsMatch(expectedFileNames, actualFileNames, "comparing expected and actual output files, expected %+v got %+v", expectedFileNames, actualFileNames)

			for _, outFile := range tt.outputFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)

				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}
			req.ElementsMatch(tt.wantResIDs, actualResIDs)
		})
	}
}

func TestExcludeKubernetesPatch(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name          string
		basePath      string
		excludedPatch resid.ResId
		wantErr       bool
		inputFiles    []fileStruct
		outputFiles   []fileStruct
	}{
		{
			name:          "existsInBase",
			basePath:      "base",
			excludedPatch: resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector"),
			wantErr:       false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector-two
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector-two
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
		},
		{
			name:          "does not exist",
			basePath:      "base",
			excludedPatch: resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector"),
			wantErr:       false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta2
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector-two
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta2
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector-two
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
		},
		{
			name:          "exists in child base",
			basePath:      "base",
			excludedPatch: resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector"),
			wantErr:       false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
bases:
- ../another/base
patchesStrategicMerge:
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extension/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- anotherresource.yaml
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- myresource.yaml
bases:
- ../another/base
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extension/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
		},
		{
			name:          "already removed",
			basePath:      "base",
			excludedPatch: resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector"),
			wantErr:       false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment-test
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesStrategicMerge:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `
apiVersion: extensions/v1beta1
kind: Deployment-test
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate`,
				},
			},
		},
		{
			name:          "jsonPatch",
			basePath:      "base",
			excludedPatch: resid.NewResId(gvk.Gvk{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}, "jaeger-collector"),
			wantErr:       false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesJson6902:
- path: chart-patch.json
  target:
    group: extensions
    kind: Deployment
    name: jaeger-collector
    version: v1beta1
- path: chart-patch-2.json
  target:
    group: extensions
    kind: Deployment
    name: istio-galley-default
    version: v1beta1
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
patchesJson6902:
- target:
    group: extensions
    version: v1beta1
    kind: Deployment
    name: istio-galley-default
  path: chart-patch-2.json
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.MkdirAll(filepath.Dir(inFile.name), os.FileMode(0644)))
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			// run exclude function
			err := ExcludeKubernetesPatch(mockFs, tt.basePath, tt.excludedPatch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExcludeKubernetesPatch() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.outputFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			err = mockFs.Walk("", func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					actualFileNames = append(actualFileNames, path)
				}
				return nil
			})
			req.NoError(err, "read output /")

			req.ElementsMatch(expectedFileNames, actualFileNames, "comparing expected and actual output files, expected %+v got %+v", expectedFileNames, actualFileNames)

			for _, outFile := range tt.outputFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)

				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}
		})
	}
}

func TestUnExcludeKubernetesResource(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name             string
		basePath         string
		excludedResource string
		wantErr          bool
		outputFiles      []fileStruct
		inputFiles       []fileStruct
	}{
		{
			name:             "existsInBase",
			basePath:         "base",
			excludedResource: "/myresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- notmyresource.yaml
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
		{
			name:             "does not exist",
			basePath:         "base",
			excludedResource: "notexist-resource.yaml",
			wantErr:          true,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
		{
			name:             "exists in child base",
			basePath:         "base",
			excludedResource: "/another-base/anotherresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
bases:
- ../another/base
resources:
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
bases:
- ../another/base
resources:
- myresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "another/base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- anotherresource.yaml
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `#unused`,
				},
			},
		},
		{
			name:             "already included",
			basePath:         "base",
			excludedResource: "myresource.yaml",
			wantErr:          false,
			inputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "base/kustomization.yaml",
					data: `kind: ""
apiversion: ""
resources:
- myresource.yaml
- notmyresource.yaml
`,
				},
				{
					name: "base/myresource.yaml",
					data: `#unused`,
				},
				{
					name: "base/notmyresource.yaml",
					data: `#unused`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.MkdirAll(filepath.Dir(inFile.name), os.FileMode(0644)))
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			// run unexclude function
			if err := UnExcludeKubernetesResource(mockFs, tt.basePath, tt.excludedResource); (err != nil) != tt.wantErr {
				t.Errorf("UnExcludeKubernetesResource() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS - contents should match originals
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.outputFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			err := mockFs.Walk("", func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					actualFileNames = append(actualFileNames, path)
				}
				return nil
			})
			req.NoError(err, "read output /")

			req.ElementsMatch(expectedFileNames, actualFileNames, "comparing expected and actual output files, expected %+v got %+v", expectedFileNames, actualFileNames)

			for _, outFile := range tt.outputFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)

				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}
		})
	}
}
