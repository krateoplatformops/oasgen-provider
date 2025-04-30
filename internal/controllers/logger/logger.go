package logger

import (
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

type Logger struct {
	Verbose bool
	logging.Logger
}

var _ logging.Logger = &Logger{}

func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.Logger.Info(msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...any) {
	if l.Verbose {
		l.Logger.Debug(msg, keysAndValues...)
	}
}
func (l *Logger) WithValues(keysAndValues ...any) logging.Logger {
	return &Logger{
		Verbose: l.Verbose,
		Logger:  l.Logger.WithValues(keysAndValues...),
	}
}
