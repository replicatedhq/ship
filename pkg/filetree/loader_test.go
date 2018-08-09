package filetree

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/replicatedhq/ship/pkg/state"

	"github.com/replicatedhq/ship/pkg/testing/tmpfs"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type TestCase struct {
	Name      string                   `yaml:"name"`
	Mkdir     []string                 `yaml:"mkdir"`
	Touch     []string                 `yaml:"touch"`
	Overlays  map[string]state.Overlay `yaml:"overlays"`
	Read      string                   `yaml:"read"`
	Expect    *Node                    `yaml:"expect"`
	ExpectErr string                   `yaml:"expectErr"`
}

func loadTestCases(t *testing.T) []TestCase {
	contents, err := ioutil.ReadFile(path.Join("test-cases", "tests.yml"))
	require.NoError(t, err, "load test cases")

	cases := make([]TestCase, 1)
	err = yaml.UnmarshalStrict(contents, &cases)

	require.NoError(t, err, "unmarshal test cases")

	return cases
}

func TestAferoLoader(t *testing.T) {
	tests := loadTestCases(t)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			fs, cleanup := tmpfs.Tmpfs(t)
			defer cleanup()
			loader := aferoLoader{FS: fs}

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

			tree, err := loader.LoadTree(toRead, &state.Kustomize{
				Overlays: test.Overlays,
			})
			if test.ExpectErr == "" {
				req.NoError(err)
			} else {
				req.Regexp(test.ExpectErr, err.Error())
				return
			}

			req.True(EqualTrees(*tree, *test.Expect))
		})
	}
}

func EqualTrees(node Node, expectNode Node) bool {
	treesAreEqual := true

	if len(node.Children) == 0 && len(expectNode.Children) == 0 {
		if node.Name == expectNode.Name {
			return true
		}
		return false
	}
	if len(node.Children) != len(expectNode.Children) {
		return false
	}

	if len(node.Children) > 0 && len(expectNode.Children) > 0 && len(node.Children) == len(expectNode.Children) {
		doChildrenMatch := EqualChildren(node.Children, expectNode.Children)
		treesAreEqual = treesAreEqual && doChildrenMatch
	}

	return treesAreEqual
}

func EqualChildren(nodes []Node, expectNodes []Node) bool {
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

		doChildrenMatch := EqualTrees(node, matchingExpectNode)
		allChildrenMatch = allChildrenMatch && doChildrenMatch
	}
	return allChildrenMatch
}
