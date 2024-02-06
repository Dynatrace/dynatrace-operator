package support_archive

import (
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	supportArchiveLoggerName = "[support-archive]"
)

func newSupportArchiveLogger(out io.Writer) logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(supportArchiveLoggerName)
}

func logInfof(log logr.Logger, format string, v ...any) {
	log.V(1).Info("foobar")
	log.Info(fmt.Sprintf(format, v...))
}

func logErrorf(log logr.Logger, err error, format string, v ...any) {
	log.Error(err, fmt.Sprintf(format, v...))
}
