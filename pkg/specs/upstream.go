package specs

import (
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-version"
	"github.com/replicatedhq/ship/pkg/util"
)

func (r *Resolver) maybeCreateVersionedUpstream(upstream string) (string, error) {
	debug := log.With(level.Debug(r.Logger), "method", "maybeCreateVersionedUpstream")
	if util.IsGithubURL(upstream) {
		githubURL, err := util.ParseGithubURL(upstream, "master")
		if err != nil {
			debug.Log("event", "parseGithubURL.fail")
			return upstream, nil
		}

		parsedVersion, err := version.NewVersion(githubURL.Ref)
		if err == nil {
			return strings.Replace(upstream, parsedVersion.Original(), "{{ .UpstreamVersion }}", 1), nil
		}
	}

	return upstream, nil
}
