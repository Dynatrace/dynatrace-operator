package troubleshoot

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

type troubleshootLogger struct {
	logger logr.Logger
}

func newTroubleshootLogger(testName string) logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return logr.New(
		troubleshootLogger{
			logger: ctrlzap.New(ctrlzap.WriteTo(os.Stdout), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(testName),
		},
	)
}

//
// Troubleshoot fmt-like log wrappers
//

func logNewTestf(format string, v ...interface{}) {
	log.V(levelNewTest).Info(fmt.Sprintf(format, v...))
}

func logInfof(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func logOkf(format string, v ...interface{}) {
	log.V(levelOk).Info(fmt.Sprintf(format, v...))
}

func logErrorf(format string, v ...interface{}) {
	log.V(levelError).Info(fmt.Sprintf(format, v...))
}

func errorWithMessagef(err error, format string, v ...interface{}) error {
	message := fmt.Sprintf(format, v...)
	return fmt.Errorf("%s {\"error\": %s}", message, err.Error())
}

//
// implementation of LogSink interface
//

func (dtl troubleshootLogger) Init(info logr.RuntimeInfo) {}

func (dtl troubleshootLogger) Info(level int, msg string, keysAndValues ...interface{}) {
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

func (dtl troubleshootLogger) Enabled(level int) bool {
	return dtl.logger.Enabled()
}

func (dtl troubleshootLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	// prints stack trace
	dtl.logger.Error(err, prefixError+msg, keysAndValues...)
}

func (dtl troubleshootLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return troubleshootLogger{
		logger: dtl.logger.WithValues(keysAndValues...),
	}
}

func (dtl troubleshootLogger) WithName(name string) logr.LogSink {
	return troubleshootLogger{
		logger: dtl.logger.WithName(name),
	}
}
