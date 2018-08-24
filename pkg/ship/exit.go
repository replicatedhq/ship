package ship

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util/warnings"
)

// ExitWithError can be called if something goes wrong to print some friendly output
func (s *Ship) ExitWithError(err error) {
	if warnings.IsWarning(err) {
		s.ExitWithWarn(err)
		return
	}

	if s.Viper.GetString("log-level") == "debug" {
		s.UI.Error(fmt.Sprintf("There was an unexpected error! %+v", err))
	} else {
		s.UI.Error(fmt.Sprintf("There was an unexpected error! %v", err))
	}
	level.Warn(s.Logger).Log("event", "exit.withErr", "errorWithStack", fmt.Sprintf("%+v", err))
	s.UI.Output("")
	time.Sleep(100 * time.Millisecond) // hack, need to wait for flush output from above
	s.preserveDebugLogsOrRequestReRun()

	if !s.Viper.GetBool("no-os-exit") {
		os.Exit(1)
	}
}

// ExitWithWarn can be called if something goes wrong to print some friendly output
func (s *Ship) ExitWithWarn(err error) {
	s.UI.Warn(fmt.Sprintf("%v", errors.Cause(err)))
	os.Exit(1)
}

func (s *Ship) preserveDebugLogsOrRequestReRun() {
	debugLogFile := path.Join(constants.ShipPathInternalLog)
	// make sure it exists
	if exists, err := s.FS.Exists(debugLogFile); err != nil || !exists {
		s.UI.Info(
			"There was an error configuring the application. " +
				"Please re-run with --log-level=debug and include " +
				"the output in any support inquiries.",
		)
	} else {
		s.UI.Info(fmt.Sprintf(
			"There was an error configuring the application. "+
				"A debug log has been written to %q, please include it "+
				"in any support inquiries.",
			debugLogFile),
		)
	}
}
