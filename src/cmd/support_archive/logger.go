package support_archive

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	prefixInfo    = "    "
	prefixNewTest = "--- "
	prefixOk      = " \u221A  "
	prefixError   = " \u00D7  "
	levelNewTest  = 1
	levelOk       = 2
	levelError    = 3
)

type logCollectorLogger struct {
	logger logr.Logger
}

func newLogCollectorLogger(collectorName string) logr.Logger {

	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return logr.New(
		logCollectorLogger{
			// using stderr because we use stdout to deliver the tarball
			logger: ctrlzap.New(ctrlzap.WriteTo(os.Stderr), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(collectorName),
		},
	)
}

func logInfof(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func logErrorf(format string, v ...interface{}) {
	log.V(levelError).Info(fmt.Sprintf(format, v...))
}

func (dtl logCollectorLogger) Init(info logr.RuntimeInfo) {}

func (dtl logCollectorLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	switch level {
	case levelNewTest:
		dtl.logger.Info(prefixNewTest+msg, keysAndValues...)
	case levelOk:
		dtl.logger.Info(prefixOk+msg, keysAndValues...)
	case levelError:
		// no stack trace
		dtl.logger.Info(prefixError+msg, keysAndValues...)
	default:
		dtl.logger.Info(prefixInfo+msg, keysAndValues...)
	}
}

func (dtl logCollectorLogger) Enabled(level int) bool {
	return dtl.logger.Enabled()
}

func (dtl logCollectorLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	// prints stack trace
	dtl.logger.Error(err, prefixError+msg, keysAndValues...)
}

func (dtl logCollectorLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return logCollectorLogger{
		logger: dtl.logger.WithValues(keysAndValues...),
	}
}

func (dtl logCollectorLogger) WithName(name string) logr.LogSink {
	return logCollectorLogger{
		logger: dtl.logger.WithName(name),
	}
}
