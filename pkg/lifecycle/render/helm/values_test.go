package helm

import (
	"testing"

	"github.com/emosbaugh/yaml"
	"github.com/stretchr/testify/require"
)

func TestGetAllKeys(t *testing.T) {
	t.Run("get all keys", func(t *testing.T) {
		req := require.New(t)

		req.Equal([]interface{}(nil), getAllKeys(yaml.MapSlice{}))

		m1 := yaml.MapSlice{
			{Key: "a", Value: 5},
		}
		m2 := yaml.MapSlice{
			{Key: "b", Value: true},
		}
		m3 := yaml.MapSlice{
			{Key: "a", Value: "value"},
			{Key: "b", Value: false},
			{Key: "c", Value: nil},
		}
		req.Equal(
			[]interface{}{"a", "b", "c"},
			getAllKeys(m1, m2, m3),
		)
	})
}

func TestMergeHelmValues(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		user     string
		vendor   string
		expected string
	}{
		{
			name:     "merge, vendor values only",
			base:     "",
			user:     "",
			vendor:   "#comment line\nkey1: 1 # this is a comment\nkey2: a\n",
			expected: "#comment line\nkey1: 1 # this is a comment\nkey2: a\n",
		},

		{
			name: "merge, vendor and user values",
			base: `key1: 1
key2:
  - item1
deep_key:
  level1:
    level2:
      myvalue: 3
key3: a`,
			user: `key1: 1
key2:
  - item1
  - item2_added_by_user
deep_key:
  level1:
    level2:
      myvalue: modified-by-user-5
key3: a`,
			vendor: `key1: 1
key2:
  - item1
deep_key:
  level1:
    newkey: added-by-vendor
    level2:
      myvalue: 5
key3: modified-by-vendor`,
			expected: `key1: 1
key2:
- item1
- item2_added_by_user
deep_key:
  level1:
    newkey: added-by-vendor
    level2:
      myvalue: modified-by-user-5
key3: modified-by-vendor
`,
		},

		{
			name: "comments",
			base: "",
			user: `# user comment
key3: 4
# another user comment`,
			vendor:   "# comment prefix\nkey1: 1\n  # indented comment\n\n# empty line\nnested_key:\n  # nested comment line 1\n  # nested comment line 2\n  key2: 2 # inline comment\n  # nested comment line 3\nkey3: 3\n# comment suffix\n",
			expected: "# comment prefix\nkey1: 1\n  # indented comment\n# empty line\nnested_key:\n  # nested comment line 1\n  # nested comment line 2\n  key2: 2\n          # inline comment\n  # nested comment line 3\nkey3: 4\n# comment suffix\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			merged, err := MergeHelmValues(test.base, test.user, test.vendor, true)
			req.NoError(err)
			req.Equal(test.expected, merged)
		})
	}
}
