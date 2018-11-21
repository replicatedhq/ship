package specs

import (
	"context"
	"strings"

	"github.com/replicatedhq/ship/pkg/specs/githubclient"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
)

const (
	UpstreamVersionToken = "<latest>"
)

func (r *Resolver) MaybeResolveVersionedUpstream(ctx context.Context, upstream string, existingState state.State) (string, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveVersionedUpstream"))

	githubClient := githubclient.NewGithubClient(r.FS, r.Logger)
	debug.Log("event", "resolve latest release")
	latestReleaseVersion, err := githubClient.ResolveLatestRelease(ctx, upstream)
	if err != nil {
		if strings.Contains(upstream, UpstreamVersionToken) {
			return "", errors.Wrap(err, "resolve latest release")
		}
		return upstream, nil
	}

	maybeVersionedUpstream := strings.Replace(upstream, UpstreamVersionToken, latestReleaseVersion, 1)

	debug.Log("event", "check previous version")
	if existingState.Versioned().V1.Metadata != nil {
		previousVersion, err := version.NewVersion(existingState.Versioned().V1.Metadata.Version)
		if err != nil {
			return maybeVersionedUpstream, nil
		}

		latestVersion, err := version.NewVersion(latestReleaseVersion)
		if err != nil {
			return maybeVersionedUpstream, nil
		}

		if latestVersion.LessThan(previousVersion) {
			return "", errors.Wrap(err, "latest version less than previous")
		}
	}

	return maybeVersionedUpstream, nil
}

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
			return strings.Replace(upstream, parsedVersion.Original(), UpstreamVersionToken, 1), nil
		}
	}

	return upstream, nil
}
