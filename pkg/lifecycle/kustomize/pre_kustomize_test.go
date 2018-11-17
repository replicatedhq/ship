package kustomize

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/state"
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
		expectState []state.List
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
			expectState: []state.List{
				state.List{
					APIVersion: "v1",
					Path:       "/test/main.yml",
					Items: []state.MinimalK8sYaml{
						state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
						state.MinimalK8sYaml{
							Kind: "Service",
							Metadata: state.MinimalK8sMetadata{
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
			expectState: []state.List{},
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
			expectState: []state.List{
				{
					APIVersion: "v1",
					Path:       "/test/main.yml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
						{
							Kind: "Service",
							Metadata: state.MinimalK8sMetadata{
								Name: "jaeger-collector",
							},
						},
					},
				},
				{
					APIVersion: "v1",
					Path:       "/test/sub.yml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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

			actualLists := make([]state.List, 0)
			if currentState.Versioned().V1.Metadata != nil {
				actualLists = currentState.Versioned().V1.Metadata.Lists
			}

			req.ElementsMatch(tt.expectState, actualLists)
		})
	}
}
