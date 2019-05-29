package util

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestResolver_maybeSplitMultidocYaml(t *testing.T) {
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
	}{
		{
			name:      "one doc",
			localPath: "/test",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
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
      type: Recreate
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/main.yml",
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
      type: Recreate
`,
				},
			},
		},
		{
			name:      "multidoc",
			localPath: "/multidoc",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/multidoc/multidoc.yaml",
					data: `
#A Test Comment
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
      type: Recreate

---

apiVersion: v1
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
  - name: jaeger-collector-http
    port: 14268
    protocol: TCP
    targetPort: 14268
  - name: jaeger-collector-zipkin
    port: 9411
    protocol: TCP
    targetPort: 9411
  selector:
    jaeger-infra: collector-pod
  type: ClusterIP
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/multidoc/Deployment-jaeger-collector.yaml",
					data: `
#A Test Comment
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
      type: Recreate
`,
				},
				{
					name: "/multidoc/Service-jaeger-collector.yaml",
					data: `
apiVersion: v1
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
  - name: jaeger-collector-http
    port: 14268
    protocol: TCP
    targetPort: 14268
  - name: jaeger-collector-zipkin
    port: 9411
    protocol: TCP
    targetPort: 9411
  selector:
    jaeger-infra: collector-pod
  type: ClusterIP
`,
				},
			},
		},
		{
			name:      "no yaml",
			localPath: "/test",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/test/not-yaml.md",
					data: `
##This is not a yaml file
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/not-yaml.md",
					data: `
##This is not a yaml file
`,
				},
			},
		},
		{
			name:      "flux-account",
			localPath: "/flux-account",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/flux-account/flux-account.yaml",
					data: `
---
# The service account, cluster roles, and cluster role binding are
# only needed for Kubernetes with role-based access control (RBAC).
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: flux
  name: flux
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  labels:
    name: flux
  name: flux
rules:
  - apiGroups: ['*']
    resources: ['*']
    verbs: ['*']
  - nonResourceURLs: ['*']
    verbs: ['*']
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  labels:
    name: flux
  name: flux
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flux
subjects:
  - kind: ServiceAccount
    name: flux
    namespace: default
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/flux-account/ServiceAccount-flux.yaml",
					data: `# The service account, cluster roles, and cluster role binding are
# only needed for Kubernetes with role-based access control (RBAC).
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: flux
  name: flux`,
				},
				{
					name: "/flux-account/ClusterRole-flux.yaml",
					data: `apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  labels:
    name: flux
  name: flux
rules:
  - apiGroups: ['*']
    resources: ['*']
    verbs: ['*']
  - nonResourceURLs: ['*']
    verbs: ['*']`,
				},
				{
					name: "/flux-account/ClusterRoleBinding-flux.yaml",
					data: `apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  labels:
    name: flux
  name: flux
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flux
subjects:
  - kind: ServiceAccount
    name: flux
    namespace: default
`,
				},
			},
		},
		{
			name:      "comment-before-doc",
			localPath: "/comment-before",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/comment-before/account.yaml",
					data: `
# The service account, cluster roles, and cluster role binding are
# only needed for Kubernetes with role-based access control (RBAC).
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/comment-before/account.yaml",
					data: `
# The service account, cluster roles, and cluster role binding are
# only needed for Kubernetes with role-based access control (RBAC).
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
`,
				},
			},
		},
		{
			name:      "comment-before-multidoc",
			localPath: "/comment-before-multi",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/comment-before-multi/account.yaml",
					data: `
# A comment
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
  namespace: namespace
---
apiVersion: v1
kind: Service
metadata:
  name: svcName
  labels:
    app: svcName
  namespace: anotherNamespace
spec:
  ports:
  - name: port-name
    port: 1234
    protocol: TCP
    targetPort: 1234
  type: ClusterIP
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/comment-before-multi/ServiceAccount-name-namespace.yaml",
					data: `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
  namespace: namespace`,
				},
				{
					name: "/comment-before-multi/Service-svcName-anotherNamespace.yaml",
					data: `apiVersion: v1
kind: Service
metadata:
  name: svcName
  labels:
    app: svcName
  namespace: anotherNamespace
spec:
  ports:
  - name: port-name
    port: 1234
    protocol: TCP
    targetPort: 1234
  type: ClusterIP
`,
				},
			},
		},
		{
			name:      "multidoc-with-list",
			localPath: "/comment-before-multi",
			wantErr:   false,
			inputFiles: []fileStruct{
				{
					name: "/comment-before-multi/account.yaml",
					data: `
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
  namespace: namespace
---
apiVersion: v1
kind: List
items:
- kind: Service
  apiVersion: v1
  metadata:
    name: svcName
    labels:
      app: svcName
    namespace: anotherNamespace
  spec:
    ports:
    - name: port-name
      port: 1234
      protocol: TCP
      targetPort: 1234
    type: ClusterIP
- kind: Service
  apiVersion: v1
  metadata:
    name: svcNameTwo
    labels:
      app: svcNameTwo
    namespace: anotherNamespace
  spec:
    ports:
    - name: port-name-two
      port: 12345
      protocol: TCP
      targetPort: 12345
    type: ClusterIP
`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/comment-before-multi/ServiceAccount-name-namespace.yaml",
					data: `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    name: name
  name: name
  namespace: namespace`,
				},
				{
					name: "/comment-before-multi/Service-svcName-anotherNamespace.yaml",
					data: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: svcName
  name: svcName
  namespace: anotherNamespace
spec:
  ports:
  - name: port-name
    port: 1234
    protocol: TCP
    targetPort: 1234
  type: ClusterIP
`,
				},
				{
					name: "/comment-before-multi/Service-svcNameTwo-anotherNamespace.yaml",
					data: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: svcNameTwo
  name: svcNameTwo
  namespace: anotherNamespace
spec:
  ports:
  - name: port-name-two
    port: 12345
    protocol: TCP
    targetPort: 12345
  type: ClusterIP
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
			req.NoError(mockFs.MkdirAll(tt.localPath, os.FileMode(0644)))
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			// run split function
			if err := MaybeSplitMultidocYaml(context.Background(), mockFs, tt.localPath); (err != nil) != tt.wantErr {
				t.Errorf("Resolver.maybeSplitMultidocYaml() error = %v, wantErr %v", err, tt.wantErr)
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

		})
	}
}
