package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_replaceInJSON(t *testing.T) {
	tests := []struct {
		name string
		path string
		obj  map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "noop",
			path: "",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
			want: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
		},
		{
			name: "remove top level",
			path: "a",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
			want: map[string]interface{}{
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
		},
		{
			name: "remove nonexistent",
			path: "b",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
			want: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
		},
		{
			name: "remove second level obj",
			path: "abc.bcd",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
					"hij": "klm",
				},
			},
			want: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"hij": "klm",
				},
			},
		},
		{
			name: "remove last element of obj",
			path: "abc.bcd",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": "efg",
				},
			},
			want: map[string]interface{}{
				"a": "b",
			},
		},
		{
			name: "remove non map element",
			path: "abc.bcd",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": 57,
				},
			},
			want: map[string]interface{}{
				"a": "b",
			},
		},
		{
			name: "remove child of non map element",
			path: "abc.bcd.def",
			obj: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"bcd": 57,
					"hij": "klm",
				},
			},
			want: map[string]interface{}{
				"a": "b",
				"abc": map[string]interface{}{
					"hij": "klm",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got := replaceInJSON(tt.obj, tt.path)

			req.Equal(tt.want, got)
		})
	}
}
