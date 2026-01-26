package logd

import (
	"io"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
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
		logLevel := readLogLevelFromEnv()
		baseLogger = createLogger(NewPrettyLogWriter(), logLevel)
	})

	return baseLogger
}

func LogBaseLoggerSettings() {
	logLevel := readLogLevelFromEnv()
	baseLogger.Info("logging level", "logLevel", logLevel.String())
}

func createLogger(out io.Writer, logLevel zapcore.Level) Logger {
	// It's important to create only one "main" logd to avoid excessive memory usage, creating a full logd is rather expensive,
	// deriving other loggers by WithName is rather cheap
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.StacktraceKey = stacktraceKey

	return Logger{
		ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(logLevel)),
	}
}

func readLogLevelFromEnv() zapcore.Level {
	envLevel := os.Getenv(LogLevelEnv)

	level, err := zapcore.ParseLevel(envLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	return level
}

type Logger struct {
	logr.Logger
}

// Debug can be used for verbose output that is supposed to be  valuable for troubleshooting
func (l Logger) Debug(message string, keysAndValues ...any) {
	l.debugLog(message, keysAndValues...)
}

func (l Logger) WithName(name string) Logger {
	return Logger{l.Logger.WithName(name)}
}

func (l Logger) WithValues(keysAndValues ...any) Logger {
	return Logger{l.Logger.WithValues(keysAndValues...)}
}

func (l Logger) debugLog(message string, keysAndValues ...any) {
	l.Logger.V(debugLogLevelElevation).Info(message, keysAndValues...)
}

// Write is for implementing the io.Writer interface,
// this is meant to be used to pipe (using `log.SetOutput`) the logs from the stdlib's log library which we do not use directly
// this workaround is necessary because the Webhook starts an http.Server, where we can't set the logger directly.
func (l Logger) Write(p []byte) (n int, err error) {
	l.debugLog("stdlib log", "msg", string(p))

	return len(p), nil
}
