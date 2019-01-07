package ship

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
)

func (s *Ship) WatchAndExit(ctx context.Context) error {
	if err := s.Watch(ctx); err != nil {
		s.ExitWithError(err)
		return err
	}
	return nil
}

func (s *Ship) Watch(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "watch"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	for {
		existingState, err := s.State.TryLoad()
		if err != nil {
			return errors.Wrap(err, "load state")
		}

		// This is duped and should probably be a method on state.Manager or something
		uiPrintableStatePath := s.Viper.GetString("state-file")
		if uiPrintableStatePath == "" {
			uiPrintableStatePath = constants.StatePath
		}

		if _, noExistingState := existingState.(state.Empty); noExistingState {
			debug.Log("event", "state.missing")
			return errors.New(fmt.Sprintf(`No state found at %s, please run "ship init"`, uiPrintableStatePath))
		}

		debug.Log("event", "read.upstream")

		upstream := existingState.Upstream()
		if upstream == "" {
			return errors.New(`No current chart url found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		maybeVersionedUpstream, err := s.Resolver.MaybeResolveVersionedUpstream(ctx, upstream, existingState)
		if err != nil {
			return errors.New(`Unable to resolve versioned upstream ` + upstream)
		}

		debug.Log("event", "read.lastSHA")
		lastSHA := existingState.Versioned().V1.ContentSHA
		if lastSHA == "" {
			return errors.New(`No current SHA found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		contentSHA, err := s.Resolver.ReadContentSHAForWatch(ctx, maybeVersionedUpstream)
		if err != nil {
			return errors.Wrap(err, "read content SHA")
		}

		if contentSHA != existingState.Versioned().V1.ContentSHA {
			debug.Log(
				"event", "new sha",
				"previous", existingState.Versioned().V1.ContentSHA,
				"new", contentSHA,
			)
			s.UI.Info(fmt.Sprintf("%s has an update available", maybeVersionedUpstream))
			return nil
		}

		debug.Log(
			"event", "sha.unchanged",
			"previous", existingState.Versioned().V1.ContentSHA,
			"new", contentSHA,
			"sleeping", s.Viper.GetDuration("interval"),
		)

		noUpdateMsg := fmt.Sprintf("No update was found for %s", upstream)
		s.UI.Info(noUpdateMsg)

		if s.Viper.GetBool("exit") {
			return nil
		}

		time.Sleep(s.Viper.GetDuration("interval"))
	}
}
