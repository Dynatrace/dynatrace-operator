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

	levelNewTest     = 1
	levelSuccess     = 2
	levelWarning     = 3
	levelError       = 4
	levelNewDynakube = 5
)

type troubleshootLogger struct {
	logger  logr.Logger
	subTest bool
}

func newTroubleshootLogger(testName string, subTest bool) logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	testName = fmt.Sprintf("[%-10s] ", testName)

	return logr.New(
		troubleshootLogger{
			logger:  ctrlzap.New(ctrlzap.WriteTo(os.Stdout), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(testName),
			subTest: subTest,
		},
	)
}

func logNewTestf(format string, v ...interface{}) {
	log.V(levelNewTest).Info(fmt.Sprintf(format, v...))
}

func logNewDynakubef(format string, v ...interface{}) {
	log.V(levelNewDynakube).Info(fmt.Sprintf(format, v...))
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
	var msg string

	switch level {
	case levelNewTest:
		msg = prefixNewTest + message
	case levelSuccess:
		msg = withSuccessPrefix(message)
	case levelWarning:
		msg = withWarningPrefix(message)
	case levelError:
		// Info is used for errors to suppress printing a stacktrace
		// Printing a stacktrace would confuse people in thinking the troubleshooter crashed
		msg = withErrorPrefix(message)
	case levelNewDynakube:
		msg = message
	default:
		msg = prefixInfo + message
	}

	if dtl.subTest {
		msg = " |" + msg
	}

	dtl.logger.Info(msg, keysAndValues...)
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
