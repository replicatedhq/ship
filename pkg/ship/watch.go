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

func (s *Ship) WatchAndExit(ctx context.Context) {
	if err := s.Watch(ctx); err != nil {
		s.ExitWithError(err)
	}
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

		debug.Log("event", "read.lastSHA")
		lastSHA := existingState.Versioned().V1.ContentSHA
		if lastSHA == "" {
			return errors.New(`No current SHA found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		debug.Log("event", "fetch latest chart")
		appType, localPath, err := s.AppTypeInspector.DetermineApplicationType(ctx, upstream)
		if err != nil {
			return errors.Wrapf(err, "resolve app type for %s", upstream)
		}
		debug.Log("event", "apptype.inspect", "type", appType, "localPath", localPath)

		metadata, err := s.Resolver.ResolveBaseMetadata(upstream, localPath)
		if err != nil {
			return errors.Wrapf(err, "resolve metadata and content sha for %s", upstream)
		}
		debug.Log("event", "metadata.resolve", "sha", metadata.ContentSHA)

		if metadata.ContentSHA != existingState.Versioned().V1.ContentSHA {
			debug.Log(
				"event", "new sha",
				"previous", existingState.Versioned().V1.ContentSHA,
				"new", metadata.ContentSHA,
			)
			s.UI.Info(fmt.Sprintf("%s has an update available", upstream))
			return nil
		}

		debug.Log(
			"event", "sha.unchanged",
			"previous", existingState.Versioned().V1.ContentSHA,
			"new", metadata.ContentSHA,
			"sleeping", s.Viper.GetDuration("interval"),
		)

		time.Sleep(s.Viper.GetDuration("interval"))
	}
}
