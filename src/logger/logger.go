package logger

import (
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type DTLogger struct {
	infoLogger  logr.Logger
	errorLogger logr.Logger
}

func NewDTLogger() logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return logr.New(
		DTLogger{
			infoLogger:  ctrlzap.New(ctrlzap.WriteTo(os.Stdout), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
			errorLogger: ctrlzap.New(ctrlzap.WriteTo(&errorPrettify{}), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
		},
	)
}

func (dtl DTLogger) Init(info logr.RuntimeInfo) {}

func (dtl DTLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	dtl.infoLogger.Info(msg, keysAndValues...)
}

func (dtl DTLogger) Enabled(level int) bool {
	return dtl.infoLogger.Enabled()
}

func (dtl DTLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	dtl.errorLogger.Error(err, msg, keysAndValues...)
}

func (dtl DTLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return DTLogger{
		infoLogger:  dtl.infoLogger.WithValues(keysAndValues...),
		errorLogger: dtl.errorLogger.WithValues(keysAndValues...),
	}
}

func (dtl DTLogger) WithName(name string) logr.LogSink {
	return DTLogger{
		infoLogger:  dtl.infoLogger.WithName(name),
		errorLogger: dtl.errorLogger.WithName(name),
	}
}
