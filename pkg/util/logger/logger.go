package logger

import (
	"os"
	"sync"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const LogLevelEnv = "LOG_LEVEL"

var (
	baseLogger     logr.Logger
	baseLoggerOnce sync.Once
)

// Get returns a new, unnamed logger configured with the basics we need for operator logs which can be used as a blueprint for
// derived loggers int the operator components.
func Get() logr.Logger {
	baseLoggerOnce.Do(func() {
		// its important to create only one "main" logger to avoid excessive memory usage, creating a full logger is rather expensive,
		// deriving other loggers by WithName is rather cheap
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		config.StacktraceKey = stacktraceKey
		baseLogger = ctrlzap.New(ctrlzap.WriteTo(NewPrettyLogWriter()), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(readLogLevelFromEnv()))
	})

	return baseLogger
}

func readLogLevelFromEnv() zapcore.Level {
	envLevel := os.Getenv(LogLevelEnv)

	level, err := zapcore.ParseLevel(envLevel)
	if err != nil {
		level = zapcore.DebugLevel
	}

	return level
}
