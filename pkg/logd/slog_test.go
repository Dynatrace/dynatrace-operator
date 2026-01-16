package logd

import (
	"log/slog"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestSlog(t *testing.T) {
	t.Run("simple slog text logging", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(t.Output(), nil)).WithGroup("group").With("module", "logd")
		logger.Info("hello from slog", "", "value")
		slog.Info("default slogger", "key", "value")

		logger2 := slog.New(logger.Handler()).WithGroup("group2")
		logger2.Info("hello from slog 2", "key2", "value2")
	})

	t.Run("simple slog JSON logging", func(t *testing.T) {
		// make dynamic changing of log level possible
		var lv slog.LevelVar
		lv.Set(slog.LevelInfo)

		handlerOpts := &slog.HandlerOptions{
			AddSource: false,
			Level:     &lv,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// adjust keys to be Dynatrace Logging Technology compliant
				switch a.Key {
				case slog.TimeKey:
					// adjust timestamp format
					t := a.Value.Time()

					return slog.String("timestamp", t.Format(time.RFC3339))
				case slog.MessageKey:
					a.Key = "message"
				case slog.LevelKey:
					// this one is already DT compliant
					a.Key = slog.LevelKey
				}

				return a
			},
		}

		jsonHandler := slog.NewJSONHandler(t.Output(), handlerOpts)

		logger := slog.New(jsonHandler).With("namespace", "dynatrace", "dynakube", "my-dynakube")

		logger.Error("DK reconciler failed")
		logger.Info("starting DK reconciler")

		logger2 := slog.New(logger.Handler()).With("domain", "extensions")
		logger2.Error("EEC reconiler failed", "key2", "value2")
		logger2.Warn("EEC recon warning")
		logger2.Info("start EEC reconciler", "key2", "value2")
		logger2.Debug("EEC didn't behave well")

		lv.Set(slog.LevelDebug)
		logger2.Debug("EEC didn't behave well")
	})

	t.Run("slog with zap handler", func(t *testing.T) {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		config.StacktraceKey = stacktraceKey

		zl := ctrlzap.New(ctrlzap.WriteTo(t.Output()), ctrlzap.Encoder(zapcore.NewJSONEncoder(config)), ctrlzap.Level(zapcore.Level(-100)))
		slog.SetDefault(slog.New(logr.ToSlogHandler(zl)))

		slog.Error("Error log")
		slog.Info("Info log")
		slog.Debug("Debug log")
		/*
			Zap backend basically does not easily allow to print nice loglevels

			{"level":"error","ts":"2026-01-14T16:42:01.892+0100","caller":"logd/slog_test.go:75","msg":"Error log","stacktrace":"log/slog.(*Logger).log\n\t/home/hauti/go/go-1.24.6/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.1.linux-amd64/src/log/slog/logger.go:256\nlog/slog.Error\n\t/home/hauti/go/go-1.24.6/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.1.linux-amd64/src/log/slog/logger.go:311\ngithub.com/Dynatrace/dynatrace-operator/pkg/logd.TestSlog.func3\n\t/home/hauti/wsl-ws/op-dev/pkg/logd/slog_test.go:75\ntesting.tRunner\n\t/home/hauti/go/go-1.24.6/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.1.linux-amd64/src/testing/testing.go:1934"}
			{"level":"info","ts":"2026-01-14T16:42:01.892+0100","caller":"logd/slog_test.go:76","msg":"Info log"}
			{"level":"Level(-4)","ts":"2026-01-14T16:42:01.892+0100","caller":"logd/slog_test.go:77","msg":"Debug log"}
		*/
	})
}
