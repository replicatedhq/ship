package logger

import (
	"fmt"
	"io"
	golog "log"
	"os"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-stack/stack"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type compositeLogger struct {
	loggers []log.Logger
}

func (c *compositeLogger) Log(keyvals ...interface{}) error {
	var multiErr *multierror.Error
	for _, logger := range c.loggers {
		multiErr = multierror.Append(multiErr, logger.Log(keyvals...))
	}
	return multiErr.ErrorOrNil()
}

// New builds a logger from env using viper
func New(v *viper.Viper, fs afero.Afero) log.Logger {

	fullPathCaller := pathCaller(6)
	var stdoutLogger log.Logger //nolint:gosimple
	stdoutLogger = withFormat(viper.GetString("log-format"), os.Stdout)
	stdoutLogger = log.With(stdoutLogger, "ts", log.DefaultTimestampUTC)
	stdoutLogger = log.With(stdoutLogger, "caller", fullPathCaller)
	stdoutLogger = withLevel(stdoutLogger, v.GetString("log-level"))

	debugLogFile := path.Join(constants.ShipPathInternalLog)
	var debugLogger log.Logger
	err := fs.RemoveAll(debugLogFile)
	if err != nil {
		level.Warn(stdoutLogger).Log("msg", "failed to remove existing debug log file", "path", debugLogFile, "error", err)
		golog.SetOutput(log.NewStdlibAdapter(level.Debug(stdoutLogger)))
		return stdoutLogger
	}
	debugLogWriter, err := fs.Create(debugLogFile)
	if err != nil {
		level.Warn(stdoutLogger).Log("msg", "failed to initialize debug log file", "path", debugLogFile, "error", err)
		golog.SetOutput(log.NewStdlibAdapter(level.Debug(stdoutLogger)))
		return stdoutLogger
	}

	debugLogger = withFormat(viper.GetString("log-format"), debugLogWriter)
	debugLogger = log.With(debugLogger, "ts", log.DefaultTimestampUTC)
	debugLogger = log.With(debugLogger, "caller", fullPathCaller)
	debugLogger = withLevel(debugLogger, "debug")

	realLogger := &compositeLogger{
		loggers: []log.Logger{
			stdoutLogger,
			debugLogger,
		},
	}

	golog.SetOutput(log.NewStdlibAdapter(level.Debug(realLogger)))
	return realLogger
}

func withFormat(format string, w io.Writer) log.Logger {
	switch format {
	case "json":
		return log.NewJSONLogger(w)
	case "logfmt":
		return log.NewLogfmtLogger(w)
	default:
		return log.NewLogfmtLogger(w)
	}

}

func withLevel(logger log.Logger, lvl string) log.Logger {
	switch lvl {
	case "debug":
		return level.NewFilter(logger, level.AllowDebug())
	case "info":
		return level.NewFilter(logger, level.AllowInfo())
	case "warn":
		return level.NewFilter(logger, level.AllowWarn())
	case "error":
		return level.NewFilter(logger, level.AllowError())
	case "off":
		return level.NewFilter(logger, level.AllowNone())
	default:
		logger.Log("msg", "Unknown log level, using debug", "received", lvl)
		return level.NewFilter(logger, level.AllowDebug())
	}
}

func pathCaller(depth int) log.Valuer {
	return func() interface{} {
		return fmt.Sprintf("%+s", stack.Caller(depth))
	}
}
