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
	logger, err := logr.FromContext(ctx)
	if logger.GetSink() == nil || err != nil {
		return Get()
	}

	return Logger{logger}
}
