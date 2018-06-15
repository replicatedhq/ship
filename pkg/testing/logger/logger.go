package logger

import (
	"testing"

	"github.com/go-kit/kit/log"
)

var _ log.Logger = &TestLogger{}

type TestLogger struct {
	T *testing.T
}

func (t *TestLogger) Log(keyvals ...interface{}) error {
	t.T.Log(keyvals...)
	return nil
}
