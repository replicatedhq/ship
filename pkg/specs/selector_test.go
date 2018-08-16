package specs

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalSelector(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want *Selector
	}{
		{
			name: "replicated.app",
			url:  "replicated.app?customer_id=123&installation_id=456&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "123",
				InstallationID: "456",
				ReleaseSemver:  "7.8.9",
			},
		},
		{
			name: "staging.replicated.app",
			url:  "staging.replicated.app?customer_id=123&installation_id=456&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "123",
				InstallationID: "456",
				ReleaseSemver:  "7.8.9",
				Upstream:       "https://pg.staging.replicated.com/graphql",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			parsed, err := url.Parse(test.url)
			req.NoError(err)
			actual := (&Selector{}).unmarshalFrom(parsed)
			req.Equal(test.want, actual)
		})
	}
}
