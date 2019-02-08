package util

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type kustomizeTestFile struct {
	kustomizedFile PostKustomizeFile
	contents       string
}

func (k kustomizeTestFile) toPostKustomizeFile() (PostKustomizeFile, error) {
	var out interface{}
	if err := yaml.Unmarshal([]byte(k.contents), &out); err != nil {
		return PostKustomizeFile{}, err
	}

	return PostKustomizeFile{
		Minimal: k.kustomizedFile.Minimal,
		Full:    out,
	}, nil
}

func TestRebuildListyaml(t *testing.T) {
	tests := []struct {
		name            string
		lists           []List
		kustomizedFiles []kustomizeTestFile
		expectFiles     []kustomizeTestFile
	}{
		{
			name: "single list",
			lists: []List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `
bye: goodbye
`,
				},
			},
			expectFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "List",
						},
					},
					contents: `apiVersion: v1
kind: List
items:
- hi: hello
- bye: goodbye
`,
				},
			},
		},
		{
			name:  "no list",
			lists: []List{},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `hi: hello`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `bye: goodbye`,
				},
			},
			expectFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `hi: hello
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `bye: goodbye
`,
				},
			},
		},
		{
			name: "single list with other yaml",
			lists: []List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `
bye: goodbye
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "monkey",
							},
						},
					},
					contents: `
icecream: great
`,
				},
			},
			expectFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "List",
						},
					},
					contents: `apiVersion: v1
kind: List
items:
- hi: hello
- bye: goodbye
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "monkey",
							},
						},
					},
					contents: `icecream: great
`,
				},
			},
		},
		{
			name: "multiple lists with other yaml",
			lists: []List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
				{
					APIVersion: "v1",
					Path:       "test/icecream.yaml",
					Items: []MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "chocolate",
							},
						},
						{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "strawberry",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `
bye: goodbye
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "strawberry",
							},
						},
					},
					contents: `
icecream: great
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "chocolate",
							},
						},
					},
					contents: `
cookies: wow
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "ghost",
							},
						},
					},
					contents: `
mint: chocolate
`,
				},
			},
			expectFiles: []kustomizeTestFile{
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "List",
						},
					},
					contents: `apiVersion: v1
kind: List
items:
- hi: hello
- bye: goodbye
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "List",
						},
					},
					contents: `apiVersion: v1
kind: List
items:
- cookies: wow
- icecream: great
`,
				},
				{
					kustomizedFile: PostKustomizeFile{
						Minimal: MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: MinimalK8sMetadata{
								Name: "ghost",
							},
						},
					},
					contents: `mint: chocolate
`,
				},
			},
		},
		{
			name: "empty list",
			lists: []List{
				{
					APIVersion: "v1",
					Path:       "test/empty.yaml",
					Items:      []MinimalK8sYaml{},
				},
			},
			kustomizedFiles: []kustomizeTestFile{},
			expectFiles:     []kustomizeTestFile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			testLogger := &logger.TestLogger{T: t}

			kustomizeFiles := make([]PostKustomizeFile, 0)
			for _, kustomizeTestFile := range tt.kustomizedFiles {
				actualPostKustomizeFile, err := kustomizeTestFile.toPostKustomizeFile()
				req.NoError(err)
				kustomizeFiles = append(kustomizeFiles, actualPostKustomizeFile)
			}

			rebuilt, err := RebuildListYaml(testLogger, tt.lists, kustomizeFiles)
			req.NoError(err)

			actualContents := make([]string, 0)
			actualMinimal := make([]MinimalK8sYaml, 0)
			expectedContents := make([]string, 0)
			expectedMinimal := make([]MinimalK8sYaml, 0)
			for idx, rebuiltFile := range rebuilt {
				rebuiltFileB, err := yaml.Marshal(rebuiltFile.Full)
				req.NoError(err)

				actualContents = append(actualContents, string(rebuiltFileB))
				actualMinimal = append(actualMinimal, rebuiltFile.Minimal)

				expectedContents = append(expectedContents, string(tt.expectFiles[idx].contents))
				expectedMinimal = append(expectedMinimal, tt.expectFiles[idx].kustomizedFile.Minimal)
			}
			req.ElementsMatch(expectedContents, actualContents)
			req.ElementsMatch(expectedMinimal, actualMinimal)
		})
	}
}
