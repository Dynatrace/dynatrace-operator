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
		baseLogger.Info("logging level", "logLevel", logLevel.String())
	})

	return baseLogger
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
func (l *Logger) Debug(message string, keysAndValues ...any) {
	kv := make([]any, 0)
	kv = append(kv, keysAndValues...)

	l.debugLog(message, kv...)
}

func (l Logger) WithName(name string) Logger {
	return Logger{l.Logger.WithName(name)}
}

func (l Logger) WithValues(keysAndValues ...any) Logger {
	return Logger{l.Logger.WithValues(keysAndValues...)}
}

func (l *Logger) debugLog(message string, keysAndValues ...any) {
	l.Logger.V(debugLogLevelElevation).Info(message, keysAndValues...)
}
