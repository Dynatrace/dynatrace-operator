package logd

import (
	"fmt"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
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
		switch int(l) {

		// TODO: our debug levels and zaplevels overlap inconsistently, fix it
		case int(TraceLevel) * -1:
			enc.AppendString("trace")
		case int(DebugLevel) * -1:
			enc.AppendString("debug")
		case int(InfoLevel) * -1:
			enc.AppendString("info")
		case int(WarnLevel) * -1:
			enc.AppendString("warn")
		case int(zapcore.DebugLevel):
			enc.AppendString("debug")
		case int(zapcore.ErrorLevel):
			// errors are handled differently
			enc.AppendString("error")
		case int(zapcore.PanicLevel):
			enc.AppendString("panic")
		case int(zapcore.DPanicLevel):
			enc.AppendString("dpanic")
		case int(zapcore.WarnLevel):
			enc.AppendString("warn")
		case int(zapcore.InfoLevel):
			enc.AppendString("info")

		default:
			enc.AppendString(fmt.Sprintf("unknown (%d)", int(l)))
		}
	}

	//encoder := zapcore.NewConsoleEncoder(config)
	encoder := zapcore.NewJSONEncoder(config)

	// This causes that the whole DK status is written to the log lines
	/*	encoder := &ctrlzap.KubeAwareEncoder{
		Encoder: zapcore.NewJSONEncoder(config),
		Verbose: false,
	}*/

	return ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(encoder), ctrlzap.Level(LevelEnabler{}))
}

type LevelEnabler struct {
	zapcore.LevelEnabler
	zapLevel zapcore.Level
	level    LogLevel
}

func (le LevelEnabler) Enabled(level zapcore.Level) bool {
	lvl := int(level) * -1

	if lvl >= LogLevelOffset {
		return lvl >= int(le.level)
	}
	if lvl >= int(le.zapLevel) {
		return lvl >= int(le.zapLevel)
	}
	return true
}
