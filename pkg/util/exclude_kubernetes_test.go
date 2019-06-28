package util

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
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
`,
				},
				{
					name: "another/base/anotherresource.yaml",
					data: `#unused`,
				},
			},
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
			if err := ExcludeKubernetesResource(mockFs, tt.basePath, tt.excludedResource); (err != nil) != tt.wantErr {
				t.Errorf("ExcludeKubernetesResource() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.outputFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			err := mockFs.Walk("", func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					actualFileNames = append(actualFileNames, path)
					fmt.Println(path)
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
					fmt.Println(path)
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
