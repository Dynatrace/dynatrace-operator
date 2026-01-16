package logd

import (
	"context"

	"github.com/go-logr/logr"
)

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

func Error(log logr.Logger, err error, msg string, keysAndValues ...any) {
	log.V(int(ErrorLevel)).Error(err, msg, keysAndValues...)
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

func (l Logger) Error(err error, msg string, keysAndValues ...any) {
	Error(l.Logger, err, msg, keysAndValues...)
}

func (l Logger) Warn(msg string, keysAndValues ...any) {
	Warn(l.Logger, msg, keysAndValues...)
}

func (l Logger) Info(msg string, keysAndValues ...any) {
	Info(l.Logger, msg, keysAndValues...)
}

func (l Logger) Debug(msg string, keysAndValues ...any) {
	Debug(l.Logger, msg, keysAndValues...)
}

func (l Logger) Trace(msg string, keysAndValues ...any) {
	Trace(l.Logger, msg, keysAndValues...)
}

func (l Logger) Enter(msg string, keysAndValues ...any) {
	if config.LogEnterExits {
		funcName := getCaller()
		l.WithValues("func", funcName).Info(msg, keysAndValues...)
	}
}

func (l Logger) Exit(msg string, keysAndValues ...any) {
	if config.LogEnterExits {
		funcName := getCaller()
		l.WithValues("func", funcName).Info(msg, keysAndValues...)
	}
}

func getCaller() string {
	// todo
	return "foobar"
}
