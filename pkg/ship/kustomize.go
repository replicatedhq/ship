package ship

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

func (s *Ship) Init(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "init"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)
	removeExistingState := !s.Viper.GetBool("preserve-state")

	if os.Getenv("PRELOAD_TEST_STATE") == "1" {
		if err := s.maybeWriteStateFromFile(); err != nil {
			return err
		}
	}

	if s.Viper.GetString("raw") != "" {
		release := s.fakeKustomizeRawRelease()
		return s.execute(ctx, release, nil)
	}

	if s.Viper.GetString("helm-values-file") != "" {
		_, err := filepath.Abs(s.Viper.GetString("helm-values-file"))
		if err != nil {
			return warnings.WarnFileNotFound(s.Viper.GetString("helm-values-file"))
		}
	}

	existingState, _ := s.State.CachedState()
	if !existingState.IsEmpty() && s.Viper.GetString("state-file") == "" {
		debug.Log("event", "existing.state")

		if s.Viper.GetString("state-from") != "file" {
			debug.Log("event", "existing.state", "state-from", "not file")
			return warnings.WarnCannotRemoveState
		}

		if removeExistingState {
			if err := s.promptToRemoveState(); err != nil {
				debug.Log("event", "state.remove.prompt.fail")
				return err
			}
		} else {
			s.UI.Info("Preserving current state")
			if !s.upstreamMatchesExisting(existingState) {
				return errors.New(fmt.Sprintf("Upstream %s does not match upstream from state %s", s.Viper.GetString("upstream"), existingState.Upstream()))
			}
		}
	}

	if removeExistingState && os.Getenv("PRELOAD_TEST_STATE") != "1" {
		if err := s.maybeWriteStateFromFile(); err != nil {
			return err
		}
	}

	s.State.UpdateVersion()

	// we already check in the CMD, but no harm in being extra safe here
	target := s.Viper.GetString("upstream")
	if target == "" {
		return errors.New("No upstream provided")
	}

	p := s.Resolver.NewContentProcessor()
	maybeVersionedUpstream, err := p.MaybeResolveVersionedUpstream(ctx, target, existingState)
	if err != nil {
		return errors.Wrap(err, "create versioned release")
	}

	release, err := s.Resolver.ResolveRelease(ctx, maybeVersionedUpstream)
	if err != nil {
		return errors.Wrap(err, "resolve release")
	}

	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)
	if err := s.execute(ctx, release, nil); err != nil {
		return errors.Wrap(err, "execute")
	}

	if err := s.State.CommitState(); err != nil {
		return errors.Wrap(err, "commit state")
	}

	return nil
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

func (s *Ship) upstreamMatchesExisting(existing state.State) bool {
	currentUpstream := s.Viper.GetString("upstream")
	existingUpstream := existing.Upstream()
	return currentUpstream == existingUpstream
}
