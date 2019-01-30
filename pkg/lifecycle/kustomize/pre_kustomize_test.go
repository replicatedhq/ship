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
)

func Test_maybeSplitListYaml(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name        string
		localPath   string
		wantErr     bool
		inputFiles  []fileStruct
		outputFiles []fileStruct
		expectState []util.List
	}{
		{
			name:      "single list with two items",
			localPath: "/test",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `
apiVersion: v1
kind: List
items:
- apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    name: jaeger-collector
    labels:
      app: jaeger
      jaeger-infra: collector-deployment
    spec:
      replicas: 1
      strategy:
        type: Recreate
- apiVersion: v1
  kind: Service
  metadata:
    name: jaeger-collector
    labels:
      app: jaeger
      jaeger-infra: collector-service
  spec:
    ports:
    - name: jaeger-collector-tchannel
      port: 14267
      protocol: TCP
      targetPort: 14267
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/Deployment-jaeger-collector.yaml",
					data: `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  name: jaeger-collector
  spec:
    replicas: 1
    strategy:
      type: Recreate
`,
				},
				{
					name: "/test/Service-jaeger-collector.yaml",
					data: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: jaeger
    jaeger-infra: collector-service
  name: jaeger-collector
spec:
  ports:
  - name: jaeger-collector-tchannel
    port: 14267
    protocol: TCP
    targetPort: 14267
`,
				},
			},
			expectState: []util.List{
				util.List{
					APIVersion: "v1",
					Path:       "/test/main.yml",
					Items: []util.MinimalK8sYaml{
						util.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: util.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
						util.MinimalK8sYaml{
							Kind: "Service",
							Metadata: util.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
					},
				},
			},
		},
		{
			name:      "not a list",
			localPath: "/test",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: jaeger-collector
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate
`,
				},
			},
			expectState: []util.List{},
		},
		{
			name:      "multiple lists",
			localPath: "/test",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `
apiVersion: v1
kind: List
items:
- apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    name: jaeger-collector
    labels:
      app: jaeger
      jaeger-infra: collector-deployment
    spec:
      replicas: 1
      strategy:
        type: Recreate
- apiVersion: v1
  kind: Service
  metadata:
    name: jaeger-collector
    labels:
      app: jaeger
      jaeger-infra: collector-service
  spec:
    ports:
    - name: jaeger-collector-tchannel
      port: 14267
      protocol: TCP
      targetPort: 14267
`,
				},
				{
					name: "/test/sub.yml",
					data: `
apiVersion: v1
kind: List
items:
- apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    name: jaeger-query
    labels:
      app: jaeger
      jaeger-infra: query-deployment
  spec:
    replicas: 1
    strategy:
      type: Recreate
    template:
      metadata:
        labels:
          app: jaeger
          jaeger-infra: query-pod
        annotations:
          prometheus.io/scrape: "true"
          prometheus.io/port: "16686"
      spec:
        containers:
        - image: jaegertracing/jaeger-query:1.7.0
          name: jaeger-query
          args: ["--config-file=/conf/query.yaml"]
          ports:
          - containerPort: 16686
            protocol: TCP
          readinessProbe:
            httpGet:
              path: "/"
              port: 16687
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/Deployment-jaeger-collector.yaml",
					data: `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: jaeger
    jaeger-infra: collector-deployment
  name: jaeger-collector
  spec:
    replicas: 1
    strategy:
      type: Recreate
`,
				},
				{
					name: "/test/Service-jaeger-collector.yaml",
					data: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: jaeger
    jaeger-infra: collector-service
  name: jaeger-collector
spec:
  ports:
  - name: jaeger-collector-tchannel
    port: 14267
    protocol: TCP
    targetPort: 14267
`,
				},
				{
					name: "/test/Deployment-jaeger-query.yaml",
					data: `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: jaeger
    jaeger-infra: query-deployment
  name: jaeger-query
spec:
  replicas: 1
  strategy:
    type: Recreate
  template:
    metadata:
      annotations:
        prometheus.io/port: "16686"
        prometheus.io/scrape: "true"
      labels:
        app: jaeger
        jaeger-infra: query-pod
    spec:
      containers:
      - args:
        - --config-file=/conf/query.yaml
        image: jaegertracing/jaeger-query:1.7.0
        name: jaeger-query
        ports:
        - containerPort: 16686
          protocol: TCP
        readinessProbe:
          httpGet:
            path: /
            port: 16687
`,
				},
			},
			expectState: []util.List{
				{
					APIVersion: "v1",
					Path:       "/test/main.yml",
					Items: []util.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: util.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
						{
							Kind: "Service",
							Metadata: util.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
					},
				},
				{
					APIVersion: "v1",
					Path:       "/test/sub.yml",
					Items: []util.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: util.MinimalK8sMetadata{
								Name: "jaeger-query",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			req.NoError(mockFs.MkdirAll(tt.localPath, os.FileMode(0644)))
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			l := Kustomizer{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
				State: &state.MManager{
					Logger: log.NewNopLogger(),
					FS:     mockFs,
					V:      viper.New(),
				},
			}

			// run split function
			if err := l.maybeSplitListYaml(context.Background(), tt.localPath); (err != nil) != tt.wantErr {
				t.Errorf("Kustomizer.maybeSplitListYaml() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			filesList, err := mockFs.ReadDir(tt.localPath)
			req.NoError(err, "read output dir %s", tt.localPath)
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.outputFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			for _, actualFile := range filesList {
				actualFileNames = append(actualFileNames, filepath.Join(tt.localPath, actualFile.Name()))
			}

			req.ElementsMatch(expectedFileNames, actualFileNames, "comparing expected and actual output files, expected %+v got %+v", expectedFileNames, actualFileNames)

			for _, outFile := range tt.outputFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)
				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}

			currentState, err := l.State.TryLoad()
			req.NoError(err)

			actualLists := make([]util.List, 0)
			if currentState.Versioned().V1.Metadata != nil {
				actualLists = currentState.Versioned().V1.Metadata.Lists
			}

			req.ElementsMatch(tt.expectState, actualLists)
		})
	}
}

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
		built    []postKustomizeFile
		original []testFile
		expect   []testFile
		wantErr  bool
	}{
		{
			name: "replace single file",
			step: api.Kustomize{
				Base: "",
			},
			built: []postKustomizeFile{
				{
					minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "strawberry",
						},
					},
					full: map[string]interface{}{
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
			built: []postKustomizeFile{
				{
					minimal: util.MinimalK8sYaml{
						Kind: "CustomResourceDefinition",
						Metadata: util.MinimalK8sMetadata{
							Name: "strawberry",
						},
					},
					full: map[string]interface{}{
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
			built: []postKustomizeFile{
				{
					minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "banana",
						},
					},
					full: map[string]interface{}{
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
			built: []postKustomizeFile{
				{
					minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "dragonfruit",
						},
					},
					full: map[string]interface{}{
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
					minimal: util.MinimalK8sYaml{
						Kind: "Fruit",
						Metadata: util.MinimalK8sMetadata{
							Name: "pomegranate",
						},
					},
					full: map[string]interface{}{
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
