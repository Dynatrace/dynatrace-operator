package logd

import (
	"fmt"
	"github.com/go-logr/logr"
	"log/slog"
	"strings"
)

type LogLevel slog.Level

// consts for zap

const (
	LogLevelOffset = 100
	// these values are chosen to correspond with zapcore.Level values
	TraceLevel LogLevel = LogLevelOffset + 3
	DebugLevel LogLevel = LogLevelOffset + 2
	InfoLevel  LogLevel = LogLevelOffset + 1
	WarnLevel  LogLevel = LogLevelOffset + 0
	ErrorLevel LogLevel = -1

	DefaultLevel = InfoLevel
)

// consts for slog
/*
const (
	// these values are chosen to correspond with zapcore.Level values
	TraceLevel LogLevel = 3
	DebugLevel LogLevel = 2
	InfoLevel  LogLevel = 1
	WarnLevel  LogLevel = 0
	ErrorLevel LogLevel = -8

	DefaultLevel = InfoLevel
)*/

func (l LogLevel) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "":
		return DefaultLevel, nil
	default:
		return DefaultLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

func Error(log logr.Logger, err error, msg string, keysAndValues ...any) {
	// error logs can't be blocked, no log level check done in logrus
	log.Error(err, msg, keysAndValues...)
}

func Warn(log logr.Logger, msg string, keysAndValues ...any) {
	log.V(int(WarnLevel)).Info(msg, keysAndValues...)
}

func Info(log logr.Logger, msg string, keysAndValues ...any) {
	log.V(int(InfoLevel)).Info(msg, keysAndValues...)
}

func Debug(log logr.Logger, msg string, keysAndValues ...any) {
	log.V(int(DebugLevel)).Info(msg, keysAndValues...)
}

func Trace(log logr.Logger, msg string, keysAndValues ...any) {
	log.V(int(TraceLevel)).Info(msg, keysAndValues...)
}
