package logger

import (
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type logSink struct {
	infoLogger  logr.Logger
	errorLogger logr.Logger
}

func newLogger() logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return logr.New(
		logSink{
			infoLogger:  ctrlzap.New(ctrlzap.WriteTo(os.Stdout), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
			errorLogger: ctrlzap.New(ctrlzap.WriteTo(&errorPrettify{}), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
		},
	)
}

func (dtl logSink) Init(logr.RuntimeInfo) {}

func (dtl logSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	dtl.infoLogger.Info(msg, keysAndValues...)
}

func (dtl logSink) Enabled(int) bool {
	return dtl.infoLogger.Enabled()
}

func (dtl logSink) Error(err error, msg string, keysAndValues ...interface{}) {
	dtl.errorLogger.Error(err, msg, keysAndValues...)
}

func (dtl logSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return logSink{
		infoLogger:  dtl.infoLogger.WithValues(keysAndValues...),
		errorLogger: dtl.errorLogger.WithValues(keysAndValues...),
	}
}

func (dtl logSink) WithName(name string) logr.LogSink {
	return logSink{
		infoLogger:  dtl.infoLogger.WithName(name),
		errorLogger: dtl.errorLogger.WithName(name),
	}
}
