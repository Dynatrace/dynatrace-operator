package logger

import (
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const LogLevelEnv = "LOG_LEVEL"

// Get returns a new, unnamed logger configured with the basics we need for operator logs which can be used as a blueprint for
// derived loggers int the operator components.
func Get() logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.StacktraceKey = stacktraceKey
	logger := ctrlzap.New(ctrlzap.WriteTo(NewPrettyLogWriter()), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(readLogLevelFromEnv()))
	return logger
}

func readLogLevelFromEnv() zapcore.Level {
	envLevel := os.Getenv(LogLevelEnv)
	level, err := zapcore.ParseLevel(envLevel)
	if err != nil {
		level = zapcore.DebugLevel
	}
	return level
}
