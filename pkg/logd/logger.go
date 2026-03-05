package logd

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

const (
	LogLevelEnv            = "LOG_LEVEL"
	debugLogLevelElevation = 1
)

var (
	baseLogger     logr.Logger
	baseLoggerOnce sync.Once
)

// Get returns a new, unnamed logd configured with the basics we need for operator logs which can be used as a blueprint for
// derived loggers int the operator components.
func Get() logr.Logger {
	baseLoggerOnce.Do(func() {
		verbosity := readVerbosityFromEnv()
		baseLogger = createLogger(NewPrettyLogWriter(), verbosity)
	})

	return baseLogger
}

func LogBaseLoggerSettings() {
	verbosity := readVerbosityFromEnv()
	baseLogger.Info("logging level", "verbosity", verbosity)
}

func createLogger(out io.Writer, verbosity int) logr.Logger {
	// Initialize klog with custom settings
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)
	// Set verbosity level
	_ = fs.Set("v", strconv.Itoa(verbosity))
	// Disable logging to stderr by default (we'll use our custom writer)
	_ = fs.Set("logtostderr", "false")
	_ = fs.Set("alsologtostderr", "false")

	// Create a JSON formatter wrapper around the output writer
	jsonWriter := &jsonLogWriter{out: out}
	klog.SetOutput(jsonWriter)

	// Create logr.Logger from klog
	return klog.NewKlogr()
}

func readVerbosityFromEnv() int {
	envLevel := os.Getenv(LogLevelEnv)

	// Map traditional log levels to verbosity levels
	// info -> 0, debug -> 1, trace -> 2, etc.
	switch envLevel {
	case "debug":
		return 1
	case "trace":
		return 2
	case "extended":
		return 3
	case "verbose":
		return 4
	case "info", "":
		return 0
	default:
		// Try to parse as integer verbosity level
		if v, err := strconv.Atoi(envLevel); err == nil && v >= 0 {
			return v
	}

		return 0
	}
}

// LogWriter wraps a logr.Logger to implement io.Writer interface
// This is meant to be used to pipe logs from the stdlib's log library
// which we do not use directly. This workaround is necessary because
// the Webhook starts an http.Server, where we can't set the logger directly.
type LogWriter struct {
	Logger logr.Logger
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	w.Logger.V(debugLogLevelElevation).Info("stdlib log", "msg", string(p))
	return len(p), nil
}

// jsonLogWriter wraps an io.Writer to format klog output as JSON
type jsonLogWriter struct {
	out io.Writer
}

func (w *jsonLogWriter) Write(p []byte) (n int, err error) {
	// klog output format is typically: "I0305 13:10:03.738107  462040 logger_test.go:105] \"message\" key1=\"value1\""
	// Where I = Info, E = Error, W = Warning
	line := string(p)
	// Create a JSON log entry with structured fields
	logEntry := make(map[string]interface{})
	// Parse klog format to extract fields
	// Basic parsing - just wrap the klog message for now
	// A more sophisticated parser could extract level, timestamp, caller, message separately
	if len(line) > 0 && line[0] >= 'A' && line[0] <= 'Z' {
		// First character is the log level
		switch line[0] {
		case 'I':
			logEntry["level"] = "info"
		case 'E':
			logEntry["level"] = "error"
		case 'W':
			logEntry["level"] = "warning"
		default:
			logEntry["level"] = "info"
}
	} else {
		logEntry["level"] = "info"
}

	// Use current timestamp in ISO8601 format
	logEntry["ts"] = time.Now().Format(time.RFC3339)
	// For now, store the entire klog line as msg
	// In a production implementation, we could parse this more thoroughly
	logEntry["msg"] = line

	// Marshal to JSON
	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		// If JSON marshaling fails, write the original line
		return w.out.Write(p)
	}
	// Write JSON followed by newline
	jsonBytes = append(jsonBytes, '\n')
	return w.out.Write(jsonBytes)
}
