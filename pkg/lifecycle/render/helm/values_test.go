package helm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAllKeys(t *testing.T) {
	t.Run("get all keys", func(t *testing.T) {
		req := require.New(t)

		req.Equal([]string{}, getAllKeys(map[string]interface{}{}))

		m1 := map[string]interface{}{"a": 5}
		m2 := map[string]interface{}{"b": true}
		m3 := map[string]interface{}{"a": "value", "b": false, "c": nil}
		allKeys := getAllKeys(m1, m2, m3)
		req.Contains(allKeys, "a")
		req.Contains(allKeys, "b")
		req.Contains(allKeys, "c")
		req.Len(allKeys, 3)
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
			vendor:   "key1: 1\nkey2: a\n",
			expected: "key1: 1\nkey2: a\n",
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

			expected: `deep_key:
  level1:
    level2:
      myvalue: modified-by-user-5
    newkey: added-by-vendor
key1: 1
key2:
- item1
- item2_added_by_user
key3: modified-by-vendor
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			merged, err := MergeHelmValues(test.base, test.user, test.vendor)
			req.NoError(err)
			req.Equal(test.expected, merged)
		})
	}
}
