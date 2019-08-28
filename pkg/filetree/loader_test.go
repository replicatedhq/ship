package filetree

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"github.com/replicatedhq/ship/pkg/state"
	state2 "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/tmpfs"
)

type TestCase struct {
	Name      string            `yaml:"name"`
	Mkdir     []string          `yaml:"mkdir"`
	Touch     []string          `yaml:"touch"`
	Patches   map[string]string `yaml:"patches"`
	Resources map[string]string `yaml:"resources"`
	Read      string            `yaml:"read"`
	Expect    *Node             `yaml:"expect"`
	ExpectErr string            `yaml:"expectErr"`
}

func loadTestCases(t *testing.T) []TestCase {
	contents, err := ioutil.ReadFile(path.Join("test-cases", "tests.yml"))
	require.NoError(t, err, "load test cases")

	cases := make([]TestCase, 1)
	err = yaml.Unmarshal(contents, &cases)

	require.NoError(t, err, "unmarshal test cases")

	return cases
}

func TestAferoLoader(t *testing.T) {
	tests := loadTestCases(t)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)

			mockState := state2.NewMockManager(mc)
			fs, cleanup := tmpfs.Tmpfs(t)
			defer cleanup()
			loader := aferoLoader{
				FS:           fs,
				StateManager: mockState,
			}

			for _, dir := range test.Mkdir {
				req.NoError(fs.MkdirAll(dir, 0777), "create dir "+dir)
			}

			for _, file := range test.Touch {
				req.NoError(fs.WriteFile(file, []byte("fake"), 0666), "write file "+file)
			}

			toRead := test.Read
			if toRead == "" {
				toRead = "/"
			}

			testPatches := make(map[string]string)
			for key, patch := range test.Patches {
				testPatches[key] = patch
			}

			testResources := make(map[string]string)
			for key, resource := range test.Resources {
				testResources[key] = resource
			}

			mockState.EXPECT().CachedState().Return(state.State{
				V1: &state.V1{
					Kustomize: &state.Kustomize{
						Overlays: map[string]state.Overlay{
							"ship": {
								Patches:   testPatches,
								Resources: testResources,
							},
						},
					},
				},
			}, nil)

			tree, err := loader.LoadTree(toRead)
			if test.ExpectErr == "" {
				req.NoError(err)
			} else {
				req.Regexp(test.ExpectErr, err.Error())
				return
			}
			eq := equalTrees(*tree, *test.Expect)

			expectTree, err := json.Marshal(test.Expect)
			req.NoError(err)
			actualTree, err := json.Marshal(tree)
			req.NoError(err)
			req.True(eq, "%s\n%s", string(expectTree), string(actualTree))
		})
	}
}

func equalTrees(node Node, expectNode Node) bool {
	treesAreEqual := true

	if len(node.Children) == 0 && len(expectNode.Children) == 0 {
		return node.Name == expectNode.Name
	}
	if len(node.Children) != len(expectNode.Children) {
		return false
	}

	if len(node.Children) > 0 && len(expectNode.Children) > 0 && len(node.Children) == len(expectNode.Children) {
		doChildrenMatch := equalChildren(node.Children, expectNode.Children)
		treesAreEqual = treesAreEqual && doChildrenMatch
	}

	return treesAreEqual
}

func equalChildren(nodes []Node, expectNodes []Node) bool {
	expectNodeMap := make(map[string]Node)
	for _, expectNode := range expectNodes {
		expectNodeMap[expectNode.Name] = expectNode
	}

	allChildrenMatch := true
	for _, node := range nodes {
		matchingExpectNode, ok := expectNodeMap[node.Name]
		if !ok {
			return false
		}

		doChildrenMatch := equalTrees(node, matchingExpectNode)
		allChildrenMatch = allChildrenMatch && doChildrenMatch
	}
	return allChildrenMatch
}
