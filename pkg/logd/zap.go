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
	/*
		config.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			switch int(l) * -1 {

			// TODO: our debug levels and zaplevels overlap inconsistently, fix it
			case int(TraceLevel):
				enc.AppendString("trace")
			case int(DebugLevel):
				enc.AppendString("debug")
			case int(InfoLevel):
				enc.AppendString("info")
			case int(WarnLevel):
				enc.AppendString("warn")
			case int(zapcore.PanicLevel) * -1:
				enc.AppendString("panic")
			case int(zapcore.DPanicLevel) * -1:
				enc.AppendString("dpanic")
			case int(zapcore.ErrorLevel) * -1:
				// errors are handled differently
				enc.AppendString("error")
			case int(zapcore.WarnLevel) * -1:
				enc.AppendString("warn")
			case int(zapcore.InfoLevel) * -1:
				enc.AppendString("info")
				//		case int(zapcore.WarnLevel) * -1:
				//			enc.AppendString("warn")

			default:
				enc.AppendString("unknown")
			}
		}
	*/
	//encoder := zapcore.NewConsoleEncoder(config)
	encoder := zapcore.NewJSONEncoder(config)

	// This causes that the whole DK status is written to the log lines
	/*	encoder := &ctrlzap.KubeAwareEncoder{
		Encoder: zapcore.NewJSONEncoder(config),
		Verbose: false,
	}*/

	return ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(encoder), ctrlzap.Level(zapcore.Level(logLevel*-1)))
}
