package support_archive

import (
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	supportArchiveLoggerName = "[support-archive]"
)

func registerSupportArchiveLogger(ctx *supportArchiveContext) {
	if ctx.toStdout {
		// using stderr because we use stdout to deliver the tarball
		ctx.log = newSupportArchiveLogger(supportArchiveLoggerName, os.Stderr)
	} else {
		ctx.log = newSupportArchiveLogger(supportArchiveLoggerName, os.Stdout)
	}
}

func newSupportArchiveLogger(name string, out io.Writer) logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""
	config.LevelKey = ""
	config.NameKey = "name"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	return ctrlzap.New(ctrlzap.WriteTo(out), ctrlzap.Encoder(zapcore.NewConsoleEncoder(config))).WithName(name)
}

func logInfof(log logr.Logger, format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func logErrorf(log logr.Logger, err error, format string, v ...interface{}) {
	log.Error(err, fmt.Sprintf(format, v...))
}
