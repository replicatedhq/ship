package util

import "testing"

func TestGenerateNameFromMetadata(t *testing.T) {
	tests := []struct {
		name    string
		k8sYaml MinimalK8sYaml
		idx     int
		want    string
	}{
		{
			name: "basic, no metadata",
			k8sYaml: MinimalK8sYaml{
				Kind: "test",
			},
			idx:  2,
			want: "test-2",
		},
		{
			name: "basic",
			k8sYaml: MinimalK8sYaml{
				Kind: "test",
				Metadata: MinimalK8sMetadata{
					Name:      "testname",
					Namespace: "testns",
				},
			},
			idx:  1,
			want: "test-testname-testns",
		},
		{
			name: "restricted characters",
			k8sYaml: MinimalK8sYaml{
				Kind: "test\\kind",
				Metadata: MinimalK8sMetadata{
					Name:      "test//name",
					Namespace: "test::ns",
				},
			},
			idx:  0,
			want: "test-kind-test--name-test--ns",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateNameFromMetadata(tt.k8sYaml, tt.idx); got != tt.want {
				t.Errorf("GenerateNameFromMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
