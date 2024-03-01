package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	baseLogger     DtLogger
	baseLoggerOnce sync.Once
)

// Get returns a new, unnamed logger configured with the basics we need for operator logs which can be used as a blueprint for
// derived loggers int the operator components.
func Get() DtLogger {
	baseLoggerOnce.Do(func() {
		logLevel := readLogLevelFromEnv()
		baseLogger = createLogger(NewPrettyLogWriter(), logLevel)
		baseLogger.Info("logging level", "logLevel", logLevel.String())
	})

	return baseLogger
}

func createLogger(out io.Writer, logLevel zapcore.Level) DtLogger {
	// its important to create only one "main" logger to avoid excessive memory usage, creating a full logger is rather expensive,
	// deriving other loggers by WithName is rather cheap
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.StacktraceKey = stacktraceKey

	return DtLogger{
		ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(logLevel)),
	}
}

func readLogLevelFromEnv() zapcore.Level {
	envLevel := os.Getenv(LogLevelEnv)

	level, err := zapcore.ParseLevel(envLevel)
	if err != nil || len(envLevel) == 0 {
		level = zapcore.DebugLevel
	}

	return level
}

type DtLogger struct {
	logr.Logger
}

// Debug can be used for verbose output that is supposed to be  valuable for troubleshooting
func (l *DtLogger) Debug(message string, keysAndValues ...any) {
	kv := make([]any, 0)
	kv = append(kv, keysAndValues...)
	kv = append(kv, "caller", getCaller())

	l.debugLog(message, kv...)
}

func (l DtLogger) WithName(name string) DtLogger {
	return DtLogger{l.Logger.WithName(name)}
}

func (l DtLogger) WithValues(keysAndValues ...any) DtLogger {
	return DtLogger{l.Logger.WithValues(keysAndValues...)}
}

func getCaller() string {
	const callerNameOffset = 2

	if pc, _, _, ok := runtime.Caller(callerNameOffset); ok {
		details := runtime.FuncForPC(pc)
		filePath, line := details.FileLine(pc)
		fileName := filepath.Base(filePath)
		functionName := filepath.Base(details.Name())

		return fmt.Sprintf("%s (%s:%d)", functionName, fileName, line)
	}

	return "<unknown function>"
}

func (l *DtLogger) DebugLogFunctionBoundaries(keysAndValues ...any) func() {
	l.debugLog("enter "+getCaller(), keysAndValues...)

	return func() {
		l.debugLog("leave "+getCaller(), keysAndValues...)
	}
}

func (l *DtLogger) debugLog(message string, keysAndValues ...any) {
	l.Logger.V(debugLogLevelElevation).Info(message, keysAndValues...)
}
