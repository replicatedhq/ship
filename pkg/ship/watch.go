package ship

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
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

		if _, noExistingState := existingState.(state.Empty); noExistingState {
			debug.Log("event", "state.missing")
			return errors.New(`No state found, please run "ship init"`)
		}

		debug.Log("event", "read.upstream")

		upstream := existingState.Upstream()
		if upstream == "" {
			return errors.New(`No current chart url found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		maybeVersionedUpstream, err := s.maybeResolveVersionedUpstream(ctx, upstream, existingState)
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

		if s.Viper.GetBool("exit") {
			noUpdateMsg := fmt.Sprintf("No update was found for %s", upstream)
			s.UI.Info(noUpdateMsg)
			return nil
		}

		time.Sleep(s.Viper.GetDuration("interval"))
	}
}
