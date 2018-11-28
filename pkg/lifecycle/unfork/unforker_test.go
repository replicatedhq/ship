package unfork

import (
	"testing"

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
