package logd

import (
	"testing"

	"github.com/go-logr/logr"
)

func TestLogging(t *testing.T) {
	t.Run("from context", func(t *testing.T) {
		ctx := logr.NewContext(t.Context(), newZapLogger(t.Output(), ErrorLevel))

		ErrorCtx(ctx, nil, "error message", "foo", "bar")
		WarnCtx(ctx, "warn message", "foo", "bar")
		InfoCtx(ctx, "info message", "foo", "bar")
		DebugCtx(ctx, "debug message", "foo", "bar")
		TraceCtx(ctx, "trace message", "foo", "bar")
	})

	t.Run("from logger", func(t *testing.T) {
		log := newZapLogger(t.Output(), TraceLevel)

		Error(log, nil, "error message", "foo", "bar")
		Warn(log, "warn message", "foo", "bar")
		Info(log, "info message", "foo", "bar")
		Debug(log, "debug message", "foo", "bar")
		Trace(log, "trace message", "foo", "bar")
	})
}
