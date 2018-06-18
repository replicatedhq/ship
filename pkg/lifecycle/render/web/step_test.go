package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestPullWeb struct {
	Name        string
	URL         string
	Method      string
	Body        string
	Headers     map[string][]string
	ExpectedErr bool
}

type TestParseRequest struct {
	Name        string
	URL         string
	Method      string
	Body        string
	ExpectedErr bool
}

func TestPullHelper(t *testing.T) {
	tests := []TestPullWeb{
		{
			Name:        "empty",
			URL:         "",
			Method:      "",
			Body:        "",
			Headers:     map[string][]string{},
			ExpectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			_, err := pullWebAsset(test.URL, test.Method, test.Body, test.Headers)
			if test.ExpectedErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}

func TestParseWebRequest(t *testing.T) {
	tests := []TestParseRequest{
		{
			Name:        "empty",
			URL:         "",
			Method:      "",
			Body:        "",
			ExpectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			_, err := parseRequest(test.URL, test.Method, test.Body)

			if test.ExpectedErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}
