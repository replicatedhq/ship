package warnings

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestStripStackIfWarning(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want string
	}{
		{
			name: "no stack",
			in:   fmt.Errorf("fake"),
			want: "fake",
		},
		{
			name: "no stack pkgerror",
			in:   errors.New("lol nope"),
			want: "lol nope",
		},
		{
			name: "pkgerror with stack",
			in:   errors.Wrap(errors.New("lol nope"), "get something"),
			want: "get something: lol nope",
		},
		{
			name: "warning without stack",
			in:   warning{msg: "lol nope"},
			want: "lol nope",
		},
		{
			name: "warning with stack",
			in:   errors.Wrap(warning{msg: "lol nope"}, "get something"),
			want: "lol nope",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, StripStackIfWarning(test.in).Error())
		})
	}
}
