package logd

import (
	"io"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newZapLogger(out io.Writer, logLevel LogLevel) logr.Logger {
	// It's important to create only one "main" logd to avoid excessive memory usage, creating a full logd is rather expensive,
	// deriving other loggers by WithName is rather cheap
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.StacktraceKey = stacktraceKey
	//	config.LevelKey = ""
	config.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch int(l) * -1 {
		case int(TraceLevel):
			enc.AppendString("trace")
		case int(DebugLevel):
			enc.AppendString("debug")
		case int(InfoLevel):
			enc.AppendString("info")
		case int(WarnLevel):
			enc.AppendString("warn")
		case int(zapcore.ErrorLevel) * -1:
			// errors are handled differently
			enc.AppendString("error")
		default:
			enc.AppendString("unknown")
		}
	}

	return ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(zapcore.Level(logLevel*-1)))
}
