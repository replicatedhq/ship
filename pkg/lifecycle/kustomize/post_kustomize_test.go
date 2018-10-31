package kustomize

import (
	"sort"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
)

type kustomizeTestFile struct {
	kustomizedFile postKustomizeFile
	contents       string
}

func (k kustomizeTestFile) toPostKustomizeFile() (postKustomizeFile, error) {
	var out interface{}
	if err := yaml.Unmarshal([]byte(k.contents), &out); err != nil {
		return postKustomizeFile{}, err
	}

	return postKustomizeFile{
		minimal: k.kustomizedFile.minimal,
		full:    out,
	}, nil
}

func TestRebuildListyaml(t *testing.T) {
	tests := []struct {
		name            string
		lists           []state.List
		kustomizedFiles []kustomizeTestFile
		expectFiles     []kustomizeTestFile
	}{
		{
			name: "single list",
			lists: []state.List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
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
			lists: []state.List{},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `hi: hello`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `bye: goodbye`,
				},
			},
			expectFiles: []kustomizeTestFile{
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `hi: hello
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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
			lists: []state.List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `
bye: goodbye
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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
			lists: []state.List{
				{
					APIVersion: "v1",
					Path:       "test/animal.yaml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
				},
				{
					APIVersion: "v1",
					Path:       "test/icecream.yaml",
					Items: []state.MinimalK8sYaml{
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "chocolate",
							},
						},
						{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "strawberry",
							},
						},
					},
				},
			},
			kustomizedFiles: []kustomizeTestFile{
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "cat",
							},
						},
					},
					contents: `
hi: hello
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "dog",
							},
						},
					},
					contents: `
bye: goodbye
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "strawberry",
							},
						},
					},
					contents: `
icecream: great
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "chocolate",
							},
						},
					},
					contents: `
cookies: wow
`,
				},
				{
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
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
					kustomizedFile: postKustomizeFile{
						minimal: state.MinimalK8sYaml{
							Kind: "Deployment",
							Metadata: state.MinimalK8sMetadata{
								Name: "ghost",
							},
						},
					},
					contents: `mint: chocolate
`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			testLogger := &logger.TestLogger{T: t}
			l := Kustomizer{
				Logger: testLogger,
			}

			kustomizeFiles := make([]postKustomizeFile, 0)
			for _, kustomizeTestFile := range tt.kustomizedFiles {
				actualPostKustomizeFile, err := kustomizeTestFile.toPostKustomizeFile()
				req.NoError(err)
				kustomizeFiles = append(kustomizeFiles, actualPostKustomizeFile)
			}

			rebuilt, err := l.rebuildListYaml(tt.lists, kustomizeFiles)
			req.NoError(err)

			actualContents := make([]string, 0)
			actualMinimal := make([]state.MinimalK8sYaml, 0)
			expectedContents := make([]string, 0)
			expectedMinimal := make([]state.MinimalK8sYaml, 0)
			for idx, rebuiltFile := range rebuilt {
				rebuiltFileB, err := yaml.Marshal(rebuiltFile.full)
				req.NoError(err)

				actualContents = append(actualContents, string(rebuiltFileB))
				actualMinimal = append(actualMinimal, rebuiltFile.minimal)

				expectedContents = append(expectedContents, string(tt.expectFiles[idx].contents))
				expectedMinimal = append(expectedMinimal, tt.expectFiles[idx].kustomizedFile.minimal)
			}
			req.ElementsMatch(expectedContents, actualContents)
			req.ElementsMatch(expectedMinimal, actualMinimal)
		})
	}
}

func TestPostKustomizeCollectionSort(t *testing.T) {
	tests := []struct {
		name             string
		collection       []postKustomizeFile
		expectCollection []postKustomizeFile
	}{
		{
			name: "sort by name",
			collection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "cat",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "apple",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "banana",
						},
					},
				},
			},
			expectCollection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "apple",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "banana",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "cat",
						},
					},
				},
			},
		},
		{
			name: "lists come first",
			collection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "hoho",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "List",
						Metadata: state.MinimalK8sMetadata{
							Name: "haha",
						},
					},
				},
			},
			expectCollection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "List",
						Metadata: state.MinimalK8sMetadata{
							Name: "haha",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name: "hoho",
						},
					},
				},
			},
		},
		{
			name: "sorts by namspace first",
			collection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name:      "chocolate",
							Namespace: "monkey",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name:      "strawberry",
							Namespace: "ape",
						},
					},
				},
			},
			expectCollection: []postKustomizeFile{
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name:      "strawberry",
							Namespace: "ape",
						},
					},
				},
				postKustomizeFile{
					minimal: state.MinimalK8sYaml{
						Kind: "Deployment",
						Metadata: state.MinimalK8sMetadata{
							Name:      "chocolate",
							Namespace: "monkey",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(postKustomizeCollection(tt.collection))
			require.Equal(t, tt.expectCollection, tt.collection)
		})
	}
}
