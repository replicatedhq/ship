package logger

import "testing"

type TestLogger struct {
	T *testing.T
}

func (l *TestLogger) Log(keyvals ...interface{}) error {
	l.T.Log(keyvals)
	return nil
}
