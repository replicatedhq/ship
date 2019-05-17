package ship

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
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

	uiPrintableStatePath := s.Viper.GetString("state-file")
	if uiPrintableStatePath == "" {
		uiPrintableStatePath = constants.StatePath
	}

	if existingState.IsEmpty() {
		debug.Log("event", "state.missing")
		return errors.Errorf(`No state file found at %s please run "ship init"`, uiPrintableStatePath)
	}

	s.State.UpdateVersion()

	debug.Log("event", "read.upstream")
	upstreamURL := existingState.Upstream()
	if upstreamURL == "" {
		return errors.Errorf(`No upstream URL found at %s, please run "ship init"`, uiPrintableStatePath)
	}

	maybeVersionedUpstream, err := s.Resolver.MaybeResolveVersionedUpstream(ctx, upstreamURL, existingState)
	if err != nil {
		return errors.New(`Unable to resolve versioned upstream ` + upstreamURL)
	}

	debug.Log("event", "fetch latest chart")
	s.Daemon.SetProgress(daemontypes.StringProgress("kustomize", `Downloading latest from upstream `+maybeVersionedUpstream))

	debug.Log("event", "reset steps completed")
	if err := s.StateManager.ResetLifecycle(); err != nil {
		return errors.Wrap(err, "reset state.json completed lifecycle")
	}

	release, err := s.Resolver.ResolveRelease(ctx, maybeVersionedUpstream)
	if err != nil {
		return errors.Wrapf(err, "resolve helm chart metadata for %s", maybeVersionedUpstream)
	}

	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)

	return s.execute(ctx, release, nil)
}
