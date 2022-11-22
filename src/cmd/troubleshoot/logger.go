package troubleshoot

import (
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
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

	levelNewCheck    = 1
	levelSuccess     = 2
	levelWarning     = 3
	levelError       = 4
	levelNewDynakube = 5
)

type troubleshootLogger struct {
	logger logr.Logger
}

type subTestLogger struct {
	troubleshootLogger
}

func newRawTroubleshootLogger(testName string, out io.Writer) troubleshootLogger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	testName = fmt.Sprintf("[%-10s] ", testName)

	return troubleshootLogger{
		logger: ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(testName),
	}
}

func newTroubleshootLoggerToWriter(testName string, out io.Writer) logr.Logger {
	return logr.New(newRawTroubleshootLogger(testName, out))
}

func newSubTestLoggerToWriter(testName string, out io.Writer) logr.Logger {
	return logr.New(subTestLogger{newRawTroubleshootLogger(testName, out)})
}

func newTroubleshootLogger(testName string) logr.Logger {
	return newTroubleshootLoggerToWriter(testName, os.Stdout)
}

func newSubTestLogger(testName string) logr.Logger {
	return newSubTestLoggerToWriter(testName, os.Stdout)
}

func logNewCheckf(format string, v ...interface{}) {
	log.V(levelNewCheck).Info(fmt.Sprintf(format, v...))
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

func (dtl troubleshootLogger) Init(_ logr.RuntimeInfo) {}

func (dtl subTestLogger) Info(level int, message string, keysAndValues ...interface{}) {
	message = addPrefixes(level, message)
	message = " |" + message
	dtl.logger.Info(message, keysAndValues...)
}

func (dtl troubleshootLogger) Info(level int, message string, keysAndValues ...interface{}) {
	message = addPrefixes(level, message)
	dtl.logger.Info(message, keysAndValues...)
}

func addPrefixes(level int, message string) string {
	switch level {
	case levelNewCheck:
		return prefixNewTest + message
	case levelSuccess:
		return withSuccessPrefix(message)
	case levelWarning:
		return withWarningPrefix(message)
	case levelError:
		// Info is used for errors to suppress printing a stacktrace
		// Printing a stacktrace would confuse people in thinking the troubleshooter crashed
		return withErrorPrefix(message)
	case levelNewDynakube:
		return message
	default:
		return prefixInfo + message
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
