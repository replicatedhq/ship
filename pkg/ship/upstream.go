package ship

import (
	"context"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/replicatedhq/ship/pkg/state"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
)

func (s *Ship) maybeResolveVersionedUpstream(ctx context.Context, upstream string, existingState state.State) (string, error) {
	debug := level.Debug(log.With(s.Logger, "method", "resolveVersionedUpstream"))

	gitClient := githubclient.NewGithubClient(s.FS, s.Logger)
	debug.Log("event", "resolve latest release")
	latestReleaseVersion, err := gitClient.ResolveLatestRelease(ctx, upstream)
	if err != nil {
		return "", errors.Wrap(err, "resolve latest release")
	}

	maybeVersionedUpstream := strings.Replace(upstream, "{{ .UpstreamVersion }}", latestReleaseVersion, 1)
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
