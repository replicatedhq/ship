package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type depGraphTestCase struct {
	dependencies map[string][]string
	resolveOrder []string
	expectError  bool

	name string
}

func TestDepGraph(t *testing.T) {
	tests := []depGraphTestCase{
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"bravo"},
				"delta":   {"alpha", "charlie"},
				"echo":    {},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			expectError:  false,
			name:         "basic_dependency_chain",
		},
		{
			dependencies: map[string][]string{
				"alpha": {"bravo"},
				"bravo": {"alpha"},
			},
			resolveOrder: []string{"alpha", "bravo"},
			expectError:  true,
			name:         "basic_circle",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"alpha"},
				"delta":   {"bravo", "charlie"},
				"echo":    {"delta"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			expectError:  false,
			name:         "basic_forked_chain",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {},
				"charlie": {"alpha"},
				"delta":   {"bravo"},
				"echo":    {"delta"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			expectError:  false,
			name:         "two_chains",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var graph depGraph
			for source, deps := range test.dependencies {
				graph.addNode(source)
				for _, dep := range deps {
					graph.addDep(source, dep)
				}
			}

			for _, toResolve := range test.resolveOrder {
				available, err := graph.getHeadNodes()
				if err != nil && test.expectError {
					return
				}

				require.NoError(t, err, "toResolve: %s", toResolve)
				require.Contains(t, available, toResolve)

				graph.resolveDep(toResolve)
			}

			available, err := graph.getHeadNodes()
			require.NoError(t, err)
			require.Empty(t, available)

			require.False(t, test.expectError, "Did not find expected error")
		})
	}
}
