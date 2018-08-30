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

	// does a state file exist on disk?
	if s.stateFileExists(ctx) {
		if err := s.promptToRemoveState(); err != nil {
			debug.Log("event", "state.remove.prompt.fail")
			return err
		}
	}

	release, err := s.Resolver.ResolveRelease(ctx, s.Viper.GetString("target"))
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
	return &api.Release{
		Spec: specs.DefaultRawRelease(s.KustomizeRaw),
	}
}
