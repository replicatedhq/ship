package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestWebAsset struct {
	Name        string
	Asset       *Built
	ExpectedErr error
}

func TestWebStep(t *testing.T) {
	tests := []TestWebAsset{
		{
			Name:        "empty",
			Asset:       nil,
			ExpectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

		})
	}
}
