package specs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type ApplyUpstreamReleaseSpec struct {
	Name             string
	Description      string
	UpstreamShipYAML string
	ExpectedSpec     *api.Spec
}

func TestSpecsResolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "specsResolver")
}

var _ = Describe("specs.Resolver", func() {

	Describe("calculateContentSHA", func() {
		Context("With multiple files", func() {
			It("should calculate the same sha, multiple times", func() {
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
				mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)

				r := Resolver{
					FS: mockFs,
					StateManager: &state.MManager{
						Logger: log.NewNopLogger(),
						FS:     mockFs,
						V:      viper.New(),
					},
				}

				firstPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				secondPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				Expect(firstPass).To(Equal(secondPass))
			})
		})
	})
})

func TestMaybeGetShipYAML(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:         "no upstream",
			Description:  "no upstream, should use default release spec",
			ExpectedSpec: nil,
		},
		{
			Name:        "upstream exists",
			Description: "upstream exists, should use upstream release spec",
			UpstreamShipYAML: `
assets:
  v1: []
config:
  v1: []
lifecycle:
  v1:
   - helmIntro: {}
`,
			ExpectedSpec: &api.Spec{
				Assets: api.Assets{
					V1: []api.Asset{},
				},
				Config: api.Config{
					V1: []libyaml.ConfigGroup{},
				},
				Lifecycle: api.Lifecycle{
					V1: []api.Step{
						{
							HelmIntro: &api.HelmIntro{},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			if test.UpstreamShipYAML != "" {
				mockFs.WriteFile(filepath.Join(constants.HelmChartPath, "ship.yaml"), []byte(test.UpstreamShipYAML), 0755)
			}

			r := Resolver{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
				ui:     cli.NewMockUi(),
			}

			ctx := context.Background()
			spec, err := r.maybeGetShipYAML(ctx, constants.HelmChartPath)
			req.NoError(err)

			req.Equal(test.ExpectedSpec, spec)
		})
	}
}

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

			r := Resolver{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
				ui:     cli.NewMockUi(),
			}

			// run split function
			if err := r.maybeSplitMultidocYaml(context.Background(), tt.localPath); (err != nil) != tt.wantErr {
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
