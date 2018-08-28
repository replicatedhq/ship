package util

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

type GithubURL struct {
	Owner  string
	Repo   string
	Ref    string
	Subdir string
}

var githubTreeRegex = regexp.MustCompile(`^[htps:/]*[w.]*github\.com/([^/?=]+)/([^/?=]+)/tree/([^/?=]+)/?(.*)$`)
var githubRegex = regexp.MustCompile(`^[htps:/]*[w.]*github\.com/([^/?=]+)/([^/?=]+)(/(.*))?$`)

func ParseGithubURL(url string, defaultRef string) (GithubURL, error) {
	var parsed GithubURL
	matches := githubTreeRegex.FindStringSubmatch(url)
	if matches != nil && len(matches) == 5 {
		parsed.Owner = matches[1]
		parsed.Repo = matches[2]
		parsed.Ref = matches[3]
		parsed.Subdir = matches[4]
	} else if matches = githubRegex.FindStringSubmatch(url); matches != nil && len(matches) == 5 {
		parsed.Owner = matches[1]
		parsed.Repo = matches[2]
		parsed.Ref = defaultRef
		parsed.Subdir = matches[4]
	}

	if parsed.Owner == "" {
		return GithubURL{}, errors.New(fmt.Sprintf("Unable to parse %q as a github url", url))
	}

	return parsed, nil
}

// returns true if this parses as a valid Github URL.
func IsGithubURL(url string) bool {
	return githubRegex.MatchString(url)
}
