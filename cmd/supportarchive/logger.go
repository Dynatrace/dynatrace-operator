package supportarchive

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

const (
	supportArchiveLoggerName = "[support-archive]"
)

func newSupportArchiveLogger(logBuffer *bytes.Buffer) logr.Logger {
	// Initialize klog with custom settings for support archive
	fs := flag.NewFlagSet("supportarchive-klog", flag.ContinueOnError)
	klog.InitFlags(fs)

	// Disable default logging to stderr
	_ = fs.Set("logtostderr", "false")
	_ = fs.Set("alsologtostderr", "false")
	_ = fs.Set("skip_headers", "true")
	// Set klog to write to both stderr and the buffer
	klog.SetOutput(io.MultiWriter(os.Stderr, logBuffer))

	// Create klog logger
	logger := klog.NewKlogr()
	return logger.WithName(supportArchiveLoggerName)
}

func logInfof(log logr.Logger, format string, v ...any) {
	log.Info(fmt.Sprintf(format, v...))
}

func logErrorf(log logr.Logger, err error, format string, v ...any) {
	log.Error(err, fmt.Sprintf(format, v...))
}
