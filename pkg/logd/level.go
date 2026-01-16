package logd

import (
	"fmt"
	"strings"
)

type LogLevel int8

const (
	TraceLevel LogLevel = 4
	DebugLevel LogLevel = 3
	InfoLevel  LogLevel = 2
	WarnLevel  LogLevel = 1
	ErrorLevel LogLevel = 0

	DefaultLevel = InfoLevel
)

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
