package replicatedapp

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
			url:  "replicated.app?customer_id=123&installation_id=456&release_id=789&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "123",
				InstallationID: "456",
				ReleaseID:      "789",
				ReleaseSemver:  "7.8.9",
			},
		},
		{
			name: "staging.replicated.app",
			url:  "staging.replicated.app?customer_id=123&installation_id=456&release_id=789&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "123",
				InstallationID: "456",
				ReleaseID:      "789",
				ReleaseSemver:  "7.8.9",
				Upstream:       "https://pg.staging.replicated.com/graphql",
			},
		},
		{
			name: "pathed app with customer id",
			url:  "replicated.app/app_id_here?customer_id=123&installation_id=456&release_id=789&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "123",
				InstallationID: "456",
				ReleaseID:      "789",
				ReleaseSemver:  "7.8.9",
			},
		},
		{
			name: "pathed app WITHOUT customer id",
			url:  "replicated.app/app_id_here?installation_id=456&release_id=789&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "app_id_here",
				InstallationID: "456",
				ReleaseID:      "789",
				ReleaseSemver:  "7.8.9",
			},
		},
		{
			name: "pathed app WITHOUT customer id and including forward slash in id",
			url:  "replicated.app/app/id/here?installation_id=456&release_id=789&release_semver=7.8.9",
			want: &Selector{
				CustomerID:     "app/id/here",
				InstallationID: "456",
				ReleaseID:      "789",
				ReleaseSemver:  "7.8.9",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			parsed, err := url.Parse(test.url)
			req.NoError(err)
			actual := (&Selector{}).UnmarshalFrom(parsed)
			req.Equal(test.want, actual)
		})
	}
}
