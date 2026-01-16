package logd

import (
	"context"
	"os"
	"sync"

	"github.com/go-logr/logr"
)

const (
	LogLevelEnv            = "LOG_LEVEL"
	debugLogLevelElevation = 1
)

var (
	baseLogger     Logger
	baseLoggerOnce sync.Once
)

// Get returns a new, unnamed logd configured with the basics we need for operator logs which can be used as a blueprint for
// derived loggers int the operator components.
func Get() Logger {
	baseLoggerOnce.Do(func() {
		logLevel, err := readLogLevelFromEnv()
		baseLogger = Logger{
			Logger: newZapLogger(NewPrettyLogWriter(), logLevel),
		}

		if err != nil {
			baseLogger.Error(err, "Failed to get log level from environment")
		}
	})

	return baseLogger
}

func FromContext(ctx context.Context) Logger {
	log := baseLogger

	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = Logger{logger}
		}
	}

	return log
}

func NewContext(ctx context.Context, logger Logger) context.Context {
	return logr.NewContext(ctx, logger.Logger)
}

func NewFromContext(ctx context.Context, name string, keysAndValues ...any) (context.Context, Logger) {
	log := FromContext(ctx).WithName(name)
	ctx = NewContext(ctx, log)

	return ctx, log
}

func LogBaseLoggerSettings() {
	logLevel, err := readLogLevelFromEnv()
	if err != nil {
		baseLogger.Error(err, "failed to read log level from environment variable")
	} else {
		baseLogger.Info("logging level", "LogLevel", logLevel.String())
	}
}

func readLogLevelFromEnv() (LogLevel, error) {
	envLevel := os.Getenv(LogLevelEnv)

	level, err := ParseLogLevel(envLevel)
	if err != nil {
		return DefaultLevel, err
	}

	return level, err
}

type Logger struct {
	logr.Logger
}

func (l Logger) WithName(name string) Logger {
	return Logger{l.Logger.WithName(name)}
}

func (l Logger) WithValues(keysAndValues ...any) Logger {
	return Logger{l.Logger.WithValues(keysAndValues...)}
}

// Write is for implementing the io.Writer interface,
// this is meant to be used to pipe (using `log.SetOutput`) the logs from the stdlib's log library which we do not use directly
// this workaround is necessary because the Webhook starts an http.Server, where we can't set the logger directly.
func (l *Logger) Write(p []byte) (n int, err error) {
	l.Debug("stdlib log", "msg", string(p))

	return len(p), nil
}
