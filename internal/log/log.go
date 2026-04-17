package log

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/oisee/vibing-steampunk/internal/config"
)

func Info(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
	}
}

func Warning(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[WARNING] "+format+"\n", args...)
	}
}

type Logger interface {
	Info(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
	Panic(format string, args ...any)
}

type StdLogger struct {
	delegate *slog.Logger
}

func (l *StdLogger) Info(format string, args ...any) {
	l.delegate.Info(format, args...)
}

func (l *StdLogger) Warning(format string, args ...any) {
	l.delegate.Warn(format, args...)
}

func (l *StdLogger) Error(format string, args ...any) {
	l.delegate.Error(format, args...)
}

func (l *StdLogger) Fatal(format string, args ...any) {
	l.delegate.Error(format, args...)
	os.Exit(1)
}

func (l *StdLogger) Panic(format string, args ...any) {
	l.delegate.Error(format, args...)
	panic(fmt.Sprintf(format, args...))
}

var logger *Logger

func GetLogger(module string) *Logger {
	if logger != nil {
		return logger
	}

	return nil
}

func Close() {
	if logger == nil {
		return
	}
}
