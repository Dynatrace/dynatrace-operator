package logd

import (
	"context"

	"github.com/go-logr/logr"
)

func NewFromContext(ctx context.Context, name string, keysAndValues ...any) (context.Context, Logger) {
	log := FromContext(ctx).WithName(name).WithValues(keysAndValues...)

	return logr.NewContext(ctx, log.Logger), log
}

func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return Get()
	}

	logger, err := logr.FromContext(ctx)
	if logger.GetSink() == nil || err != nil {
		return Get()
	}

	return Logger{logger}
}

// IntoContext stores the given Logger into ctx and returns the new context.
// Use this when explicitly seeding or replacing the logger in a context.
//
//	ctx = logd.IntoContext(ctx, myLogger)
func IntoContext(ctx context.Context, log Logger) context.Context {
	return logr.NewContext(ctx, log.Logger)
}
