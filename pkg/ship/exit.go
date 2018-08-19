package ship

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/constants"
)

// ExitWithError can be called if something goes wrong to print some friendly output
func (s *Ship) ExitWithError(err error) {
	s.printAndLogError(err, s.UI.Error)
	time.Sleep(100 * time.Millisecond) // hack, need to wait for flush output from above
	s.preserveDebugLogsOrRequestReRun()

	// we want to avoid exiting in certain integration testing scenarios
	if !s.Viper.GetBool("no-os-exit") {
		os.Exit(1)
	}
}

// ExitWithWarn can be called if something goes wrong to print some friendly output
func (s *Ship) ExitWithWarn(err error) {
	s.printAndLogError(err, s.UI.Warn)
	os.Exit(1)
}

func (s *Ship) printAndLogError(err error, uiOutput func(string)) {
	if s.Viper.GetString("log-level") == "debug" {
		uiOutput(fmt.Sprintf("There was an unexpected error! %+v", err))
	} else {
		uiOutput(fmt.Sprintf("There was an unexpected error! %v", err))
	}
	level.Warn(s.Logger).Log("event", "exit.withErr", "errorWithStack", fmt.Sprintf("%+v", err))
	s.UI.Output("")
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
