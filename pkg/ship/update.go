package ship

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
)

func (s *Ship) UpdateAndMaybeExit(ctx context.Context) error {
	if err := s.Update(ctx); err != nil {
		s.ExitWithError(err)
		return err
	}
	return nil
}

func (s *Ship) Update(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "update"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	s.Viper.Set("rm-asset-dest", true)

	s.Daemon.SetProgress(daemontypes.StringProgress("kustomize", `loading state`))
	// does a state already exist
	existingState, err := s.State.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	if _, noExistingState := existingState.(state.Empty); noExistingState {
		debug.Log("event", "state.missing")
		return errors.New(`No state file found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
	}

	debug.Log("event", "read.upstream")
	upstreamURL := existingState.Upstream()
	if upstreamURL == "" {
		return errors.New(fmt.Sprintf(`No upstream URL found at %s, please run "ship init"`, s.Viper.GetString("state-file")))
	}

	debug.Log("event", "fetch latest chart")
	s.Daemon.SetProgress(daemontypes.StringProgress("kustomize", `Downloading latest from upstream `+upstreamURL))

	release, err := s.Resolver.ResolveRelease(ctx, upstreamURL)
	if err != nil {
		return errors.Wrapf(err, "resolve helm chart metadata for %s", upstreamURL)
	}

	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)

	return s.execute(ctx, release, nil, true)
}
