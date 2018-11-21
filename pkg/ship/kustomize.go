package ship

import (
	"context"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util/warnings"
)

func (s *Ship) InitAndMaybeExit(ctx context.Context) error {
	if err := s.Init(ctx); err != nil {
		s.ExitWithError(err)
		return err
	}
	return nil
}

func (s *Ship) stateFileExists(ctx context.Context) bool {
	debug := level.Debug(log.With(s.Logger, "method", "stateFileExists"))

	existingState, err := s.State.TryLoad()
	if err != nil {
		debug.Log("event", "tryLoad.fail")
		return false
	}
	_, noExistingState := existingState.(state.Empty)

	return !noExistingState
}

func (s *Ship) Init(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "init"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	if s.Viper.GetString("raw") != "" {
		release := s.fakeKustomizeRawRelease()
		return s.execute(ctx, release, nil, true)
	}

	existingState, _ := s.State.TryLoad()
	if existingState != nil {
		if !existingState.IsEmpty() {
			debug.Log("event", "existing.state")

			if s.Viper.GetString("state-from") != "file" {
				debug.Log("event", "existing.state", "state-from", "not file")
				return warnings.WarnCannotRemoveState
			}

			if err := s.promptToRemoveState(); err != nil {
				debug.Log("event", "state.remove.prompt.fail")
				return err
			}
		}
	}

	// we already check in the CMD, but no harm in being extra safe here
	target := s.Viper.GetString("upstream")
	if target == "" {
		return errors.New("No upstream provided")
	}

	maybeVersionedUpstream, err := s.Resolver.MaybeResolveVersionedUpstream(ctx, target, existingState)
	if err != nil {
		return errors.Wrap(err, "create versioned release")
	}

	release, err := s.Resolver.ResolveRelease(ctx, maybeVersionedUpstream)
	if err != nil {
		return errors.Wrap(err, "resolve release")
	}

	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)
	return s.execute(ctx, release, nil, true)
}

func (s *Ship) promptToRemoveState() error {
	debug := level.Debug(log.With(s.Logger, "event", "promptToRemoveState"))
	debug.Log("event", "state.exists")
	if os.Getenv("RM_STATE") == "1" {
		if err := s.State.RemoveStateFile(); err != nil {
			return errors.Wrap(err, "remove existing state")
		}
	} else {
		s.UI.Warn(`
An existing .ship directory was found. If you are trying to update this application, run "ship update".
Continuing will delete this state, would you like to continue? There is no undo.`)

		useUpdate, err := s.UI.Ask(`
    Start from scratch? (y/N): `)
		if err != nil {
			return err
		}
		useUpdate = strings.ToLower(strings.Trim(useUpdate, " \r\n"))

		if useUpdate == "y" {
			// remove state.json and start from scratch
			if err := s.State.RemoveStateFile(); err != nil {
				return errors.Wrap(err, "remove existing state")
			}
		} else {
			// exit and use 'ship update'
			return warnings.WarnShouldUseUpdate
		}
	}
	return nil
}

func (s *Ship) fakeKustomizeRawRelease() *api.Release {
	r := specs.Resolver{Viper: s.Viper}
	return &api.Release{
		Spec: r.DefaultRawRelease(s.KustomizeRaw),
	}
}
