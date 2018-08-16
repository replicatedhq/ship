package specs

import (
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

// Selector selects a replicated.app spec from the Vendor's releases and channels.
// See pkg/cli/root.go for some more info on which are required and why.
//
// note that `url` struct tags are only for serialize, they don't work for deserialize
type Selector struct {
	// required
	CustomerID     string `url:"customer_id"`
	InstallationID string `url:"installation_id"`

	// optional
	Upstream      string `url:"upstream,omitempty"`
	ReleaseSemver string `url:"release_semver,omitempty"`
}

func (s *Selector) String() string {
	v, err := query.Values(s)
	if err != nil {
		return "Selector{(failed to parse)}"
	}
	return v.Encode()
}

// this is kinda janky
func (s *Selector) unmarshalFrom(url *url.URL) *Selector {
	for key, values := range url.Query() {
		switch key {
		case "customer_id":
			if len(values) != 0 {
				s.CustomerID = values[0]
			}
			continue
		case "installation_id":
			if len(values) != 0 {
				s.InstallationID = values[0]
			}
			continue
		case "release_semver":
			if len(values) != 0 {
				s.ReleaseSemver = values[0]
			}
			continue
		}
	}

	if strings.HasPrefix(url.String(), "staging.replicated.app") {
		s.Upstream = "https://pg.staging.replicated.com/graphql"
	}
	return s
}
