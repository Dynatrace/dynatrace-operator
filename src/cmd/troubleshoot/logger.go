package troubleshoot

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	prefixInfo    = "    "
	prefixNewTest = "--- "
	prefixSuccess = " \u2713  " // ✓
	prefixWarning = " \u26a0  " // ⚠
	prefixError   = " X  "      // X

	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"

	levelNewTest = 1
	levelSuccess = 2
	levelWarning = 3
	levelError   = 4
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

func logNewTestf(format string, v ...interface{}) {
	log.V(levelNewTest).Info(fmt.Sprintf(format, v...))
}

func logInfof(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func logOkf(format string, v ...interface{}) {
	log.V(levelSuccess).Info(fmt.Sprintf(format, v...))
}

func logWarningf(format string, v ...interface{}) {
	log.V(levelWarning).Info(fmt.Sprintf(format, v...))
}

func logErrorf(format string, v ...interface{}) {
	log.V(levelError).Info(fmt.Sprintf(format, v...))
}

func errorWithMessagef(err error, format string, v ...interface{}) error {
	message := fmt.Sprintf(format, v...)
	return errors.Wrapf(err, "%s {\"error\": %s}", message, err.Error())
}

func (dtl troubleshootLogger) Init(_ logr.RuntimeInfo) {}

func (dtl troubleshootLogger) Info(level int, message string, keysAndValues ...interface{}) {
	switch level {
	case levelNewTest:
		dtl.logger.Info(prefixNewTest+message, keysAndValues...)
	case levelSuccess:
		dtl.logger.Info(withSuccessPrefix(message), keysAndValues...)
	case levelWarning:
		dtl.logger.Info(withWarningPrefix(message), keysAndValues...)
	case levelError:
		// Info is used for errors to suppress printing a stacktrace
		// Printing a stacktrace would confuse people in thinking the troubleshooter crashed
		dtl.logger.Info(withErrorPrefix(message), keysAndValues...)
	default:
		dtl.logger.Info(prefixInfo+message, keysAndValues...)
	}
}

func withSuccessPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorGreen, prefixSuccess, message, colorReset)
}

func withWarningPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorYellow, prefixWarning, message, colorReset)
}

func withErrorPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorRed, prefixError, message, colorReset)
}

func (dtl troubleshootLogger) Enabled(_ int) bool {
	return dtl.logger.Enabled()
}

func (dtl troubleshootLogger) Error(err error, msg string, keysAndValues ...interface{}) {
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
