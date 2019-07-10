package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalIndent(t *testing.T) {
	veryLongLineString := "this is a very long single-line string. It should be rendered in yaml as a single line string, and not split into multiple lines. By default, go-yaml/yaml.v2 and go-yaml/yaml.v3 will split long strings onto multiple lines. This breaks some template functions, and so a PR has been opened against go-yaml/yaml.v3 to allow setting the desired length: https://github.com/go-yaml/yaml/pull/455. Until this PR is merged, we will instead be using laverya/yaml to render such strings."

	type recurse struct {
		A   int
		Rec *recurse
	}
	recurseA := recurse{A: 1}
	recurseB := recurse{A: 1, Rec: &recurseA}
	recurseA.Rec = &recurseB

	type args struct {
		indent int
		in     interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "basic struct",
			args: args{
				indent: 3,
				in: struct {
					Abc     string
					Xyz     int
					Hello   string `yaml:"world"`
					Recurse interface{}
				}{
					Abc:   "a",
					Xyz:   0,
					Hello: "this",
					Recurse: struct {
						Nested string
					}{
						Nested: "nestedstring",
					},
				},
			},
			want: `abc: a
xyz: 0
world: this
recurse:
   nested: nestedstring
`,
		},
		{
			name: "very long string line",
			args: args{
				indent: 2,
				in: struct {
					Top      string
					Indented interface{}
				}{
					Top: "top",
					Indented: struct {
						Long string
					}{
						Long: veryLongLineString,
					},
				},
			},
			want: fmt.Sprintf(`top: top
indented:
  long: '%s'
`, veryLongLineString),
		},
		{
			name: "very long multiline string line",
			args: args{
				indent: 2,
				in: struct {
					Top      string
					Indented interface{}
				}{
					Top: "top",
					Indented: struct {
						Long string
					}{
						Long: `if not split into multiple lines, this would be a rather long string
thankfully it can be split naturally where newlines already exist
as demonstrated here
otherwise things would be completely unreadable`,
					},
				},
			},
			want: `top: top
indented:
  long: |-
    if not split into multiple lines, this would be a rather long string
    thankfully it can be split naturally where newlines already exist
    as demonstrated here
    otherwise things would be completely unreadable
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := MarshalIndent(tt.args.indent, tt.args.in)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			req.Equal(tt.want, string(got))
		})
	}
}
