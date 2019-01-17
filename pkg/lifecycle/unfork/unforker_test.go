package unfork

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestContainsNonGVK(t *testing.T) {
	req := require.New(t)

	onlyGvk := `apiVersion: v1
kind: Secret
metadata:
  name: "foo"
  labels:
    something: false`

	check, err := containsNonGVK([]byte(onlyGvk))
	req.NoError(err)
	req.False(check, "yaml witih only gvk keys should not report that it contains non gvk keys")

	extraKeys := `apiVersion: v1
kind: Service
metadata:
  name: "bar"
spec:
  type: ClusterIP`

	check, err = containsNonGVK([]byte(extraKeys))
	req.NoError(err)
	req.True(check, "yaml with non gvk keys should report that it contains extra keys")
}

type testFile struct {
	path     string
	contents string
}

func TestUnforker_mapUpstream(t *testing.T) {
	tests := []struct {
		name         string
		testFiles    []testFile
		upstreamPath string
		want         map[util.MinimalK8sYaml]string
	}{
		{
			name: "simple",
			testFiles: []testFile{
				{
					path: "base/strawberry.yaml",
					contents: `kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: strawberry
spec:
  hi: hello
`,
				},
			},
			upstreamPath: "base",
			want: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "strawberry",
					},
				}: "base/strawberry.yaml",
			},
		},
		{
			name: "not native k8s",
			testFiles: []testFile{
				{
					path: "base/apple.yaml",
					contents: `kind: Fruit
apiVersion: fruity
metadata:
  name: apple
spec:
  late: night
`,
				},
			},
			upstreamPath: "base",
			want:         map[util.MinimalK8sYaml]string{},
		},
		{
			name: "complex",
			testFiles: []testFile{
				{
					path: "base/strawberry.yaml",
					contents: `kind: Service
apiVersion: v1
metadata:
  name: strawberry
spec:
  hi: hello
`,
				},
				{
					path: "base/nested/banana.yaml",
					contents: `kind: StatefulSet
apiVersion: apps/v1beta1
metadata:
  name: banana
spec:
  bye: goodbye
`,
				},
				{
					path: "base/nested/avocado.yaml",
					contents: `kind: ConfigMap
apiVersion: v1
metadata:
  name: avocado
spec:
  what: ami
`,
				},
				{
					path: "base/another/pomegranate.yaml",
					contents: `kind: Service
apiVersion: v1
metadata:
  name: pomegranate
spec:
  laugh: lol
`,
				},
			},
			upstreamPath: "base",
			want: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Service",
					Metadata: util.MinimalK8sMetadata{
						Name: "strawberry",
					},
				}: "base/strawberry.yaml",
				util.MinimalK8sYaml{
					Kind: "StatefulSet",
					Metadata: util.MinimalK8sMetadata{
						Name: "banana",
					},
				}: "base/nested/banana.yaml",
				util.MinimalK8sYaml{
					Kind: "ConfigMap",
					Metadata: util.MinimalK8sMetadata{
						Name: "avocado",
					},
				}: "base/nested/avocado.yaml",
				util.MinimalK8sYaml{
					Kind: "Service",
					Metadata: util.MinimalK8sMetadata{
						Name: "pomegranate",
					},
				}: "base/another/pomegranate.yaml",
			},
		},
	}

	for _, tt := range tests {
		req := require.New(t)
		mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
		err := addTestFiles(mockFS, tt.testFiles)
		req.NoError(err)

		t.Run(tt.name, func(t *testing.T) {
			l := &Unforker{
				Logger: &logger.TestLogger{T: t},
				FS:     mockFS,
			}
			upstreamMap := map[util.MinimalK8sYaml]string{}
			err := l.mapUpstream(upstreamMap, tt.upstreamPath)
			req.NoError(err)

			req.Equal(tt.want, upstreamMap)
		})
	}
}

func addTestFiles(fs afero.Afero, testFiles []testFile) error {
	for _, testFile := range testFiles {
		if err := fs.WriteFile(testFile.path, []byte(testFile.contents), 0777); err != nil {
			return err
		}
	}
	return nil
}

func TestUnforker_findMatchingUpstreamPath(t *testing.T) {
	tests := []struct {
		name          string
		upstreamMap   map[util.MinimalK8sYaml]string
		forkedMinimal util.MinimalK8sYaml
		want          string
	}{
		{
			name: "matching names",
			upstreamMap: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-deployment",
					},
				}: "some/deployment.yaml",
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-service",
					},
				}: "some/service.yaml",
			},
			forkedMinimal: util.MinimalK8sYaml{
				Kind: "Deployment",
				Metadata: util.MinimalK8sMetadata{
					Name: "some-deployment",
				},
			},
			want: "some/deployment.yaml",
		},
		{
			name: "forked minimal name has a prefix",
			upstreamMap: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "deployment",
					},
				}: "some/deployment.yaml",
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "service",
					},
				}: "some/service.yaml",
			},
			forkedMinimal: util.MinimalK8sYaml{
				Kind: "Deployment",
				Metadata: util.MinimalK8sMetadata{
					Name: "some-deployment",
				},
			},
			want: "some/deployment.yaml",
		},
		{
			name: "upstream resources have a prefix",
			upstreamMap: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-deployment",
					},
				}: "some/deployment.yaml",
				util.MinimalK8sYaml{
					Kind: "Service",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-service",
					},
				}: "some/service.yaml",
			},
			forkedMinimal: util.MinimalK8sYaml{
				Kind: "Service",
				Metadata: util.MinimalK8sMetadata{
					Name: "service",
				},
			},
			want: "some/service.yaml",
		},
		{
			name: "no matching resource",
			upstreamMap: map[util.MinimalK8sYaml]string{
				util.MinimalK8sYaml{
					Kind: "Deployment",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-deployment",
					},
				}: "some/deployment.yaml",
				util.MinimalK8sYaml{
					Kind: "Service",
					Metadata: util.MinimalK8sMetadata{
						Name: "some-service",
					},
				}: "some/service.yaml",
			},
			forkedMinimal: util.MinimalK8sYaml{
				Kind: "ConfigMap",
				Metadata: util.MinimalK8sMetadata{
					Name: "some-configmap",
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		req := require.New(t)
		t.Run(tt.name, func(t *testing.T) {
			l := &Unforker{}
			got := l.findMatchingUpstreamPath(tt.upstreamMap, tt.forkedMinimal)
			req.Equal(tt.want, got)
		})
	}
}
