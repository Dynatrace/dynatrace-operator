package troubleshoot

import (
	"flag"
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
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

func NewTroubleshootLoggerToWriter(out io.Writer) logr.Logger {
	// Initialize klog with custom settings for troubleshoot
	fs := flag.NewFlagSet("troubleshoot-klog", flag.ContinueOnError)
	klog.InitFlags(fs)

	// Disable default logging to stderr
	_ = fs.Set("logtostderr", "false")
	_ = fs.Set("alsologtostderr", "false")
	_ = fs.Set("skip_headers", "true")

	// Set klog to write to the specified output
	klog.SetOutput(out)

	// Create klog logger
	logger := klog.NewKlogr()

	// need to use non-empty name for root logger, otherwise name printing is omitted completely
	return logger.WithName(" ")
}

func logNewCheckf(log logr.Logger, format string, v ...any) {
	log.Info(prefixNewTest + fmt.Sprintf(format, v...))
}

func logNewDynakubef(log logr.Logger, format string, v ...any) {
	log.Info(fmt.Sprintf(format, v...))
}

func logInfof(log logr.Logger, format string, v ...any) {
	log.Info(prefixInfo + fmt.Sprintf(format, v...))
}

func logOkf(log logr.Logger, format string, v ...any) {
	log.Info(withSuccessPrefix(fmt.Sprintf(format, v...)))
}

func logWarningf(log logr.Logger, format string, v ...any) {
	log.Info(withWarningPrefix(fmt.Sprintf(format, v...)))
}

func logErrorf(log logr.Logger, format string, v ...any) {
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
