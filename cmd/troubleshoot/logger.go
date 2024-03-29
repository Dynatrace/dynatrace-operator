package troubleshoot

import (
	"fmt"
	"io"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
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

	colorError   = "\033[31m" // red
	colorOk      = "\033[32m" // green
	colorWarning = "\033[33m" // yellow
	colorReset   = "\033[0m"
)

func NewTroubleshootLoggerToWriter(out io.Writer) logd.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeName = loggerNameEncoder

	return logd.Logger{
		Logger: ctrlzap.New(
			ctrlzap.WriteTo(out),
			ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).
			// need to use non-empty name for root logd, otherwise name printing is omitted completely
			WithName(" "),
	}
}

func loggerNameEncoder(name string, encoder zapcore.PrimitiveArrayEncoder) {
	// trim space from root logd name and dot added by logr to keep only actual test name
	testName := fmt.Sprintf("[%-10s] ", strings.Trim(name, " ."))
	encoder.AppendString(testName)
}

func logNewCheckf(log logd.Logger, format string, v ...any) {
	log.Info(prefixNewTest + fmt.Sprintf(format, v...))
}

func logNewDynakubef(log logd.Logger, format string, v ...any) {
	log.Info(fmt.Sprintf(format, v...))
}

func logInfof(log logd.Logger, format string, v ...any) {
	log.Info(prefixInfo + fmt.Sprintf(format, v...))
}

func logOkf(log logd.Logger, format string, v ...any) {
	log.Info(withSuccessPrefix(fmt.Sprintf(format, v...)))
}

func logWarningf(log logd.Logger, format string, v ...any) {
	log.Info(withWarningPrefix(fmt.Sprintf(format, v...)))
}

func logErrorf(log logd.Logger, format string, v ...any) {
	log.Info(withErrorPrefix(fmt.Sprintf(format, v...)))
}

func withSuccessPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorOk, prefixSuccess, message, colorReset)
}

func withWarningPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorWarning, prefixWarning, message, colorReset)
}

func withErrorPrefix(message string) string {
	return fmt.Sprintf("%s%s%s%s", colorError, prefixError, message, colorReset)
}
