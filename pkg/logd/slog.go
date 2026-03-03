package logd

import (
	"github.com/go-logr/logr"
	"io"
	"log/slog"
	"time"
)

var logLevel slog.LevelVar

func SetLogLevel(lvl LogLevel) {
	logLevel.Set(slog.Level(lvl))
}

func newSlogger(out io.Writer, logLevel LogLevel) logr.Logger {
	// make dynamic changing of log level possible
	var lv slog.LevelVar

	// TODO: remove override
	logLevel = LogLevel(slog.LevelDebug)

	lv.Set(slog.Level(logLevel))

	handlerOpts := &slog.HandlerOptions{
		AddSource: false,
		Level:     &lv,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// adjust keys to be Dynatrace Logging Technology compliant
			switch a.Key {
			case slog.TimeKey:
				// adjust timestamp format
				t := a.Value.Time()

				return slog.String("timestamp", t.Format(time.RFC3339))
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				// this one is already DT compliant
				a.Key = slog.LevelKey
				a.Value = slogLogLeveString(a.Value)
			}

			return a
		},
	}

	jsonHandler := slog.NewJSONHandler(out, handlerOpts)

	return logr.FromSlogHandler(jsonHandler)
}

func slogLogLeveString(v slog.Value) slog.Value {
	if v.Kind() == slog.KindAny {
		switch val := v.Any().(type) {
		case slog.Level:
			return mappedLogLevelString(val)
		}
	}
	return v
}

func mappedLogLevelString(v slog.Level) slog.Value {

	if v == slog.LevelError {
		return slog.StringValue("error")
	}

	lvl := LogLevel(v.Level()) * -1
	return slog.StringValue(lvl.String())
}
