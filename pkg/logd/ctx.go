package logd

import (
	"context"
	"github.com/go-logr/logr"
)

func NewContext(ctx context.Context, logger Logger) context.Context {
	return logr.NewContext(ctx, logger.Logger)
}

func NewFromContext(ctx context.Context, name string, keysAndValues ...any) (context.Context, Logger) {
	log := FromContext(ctx).WithName(name).WithValues(keysAndValues...)
	ctx = NewContext(ctx, log)

	return ctx, log
}

func FromContext(ctx context.Context) Logger {
	log := Get()

	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = Logger{logger}
		}
	}

	return log
}

func ErrorCtx(ctx context.Context, err error, msg string, keysAndValues ...any) {
	Error(FromContext(ctx).Logger, err, msg, keysAndValues...)
}

func WarnCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Warn(FromContext(ctx).Logger, msg, keysAndValues...)
}

func InfoCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Info(FromContext(ctx).Logger, msg, keysAndValues...)
}

func DebugCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Debug(FromContext(ctx).Logger, msg, keysAndValues...)
}

func TraceCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Trace(FromContext(ctx).Logger, msg, keysAndValues...)
}
