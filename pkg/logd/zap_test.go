package logd

import (
	"errors"
	"testing"
)

func TestZap(t *testing.T) {
	t.Run("zap via wrapper", func(t *testing.T) {
		logger := Logger{
			Logger: newZapLogger(NewPrettyLogWriter(WithWriter(t.Output())), TraceLevel),
		}

		logger.Error(errors.New("Error"), "ErrorLog", "key-error", "value-error")
		logger.Warn("WarnLog", "key-warn", "value-warn")
		logger.Info("InfoLog", "key-info", "value-info")
		logger.Debug("DebugLog", "key-debug", "value-debug")
		logger.Trace("TraceLog", "key-trace", "value-trace")

		//logger.V(-1).Info("V-1Log -> V0", "key-info", "value-info")
		// these are samples how ctrl-runtime uses logs
		logger.Info("V0Log (info)")
		logger.V(1).Info("V1Log (debug)", "key-info", "value-info")
		logger.V(2).Info("V2Log (trace)", "key-info", "value-info")
		logger.V(3).Info("V3Log", "key-info", "value-info")
		logger.V(4).Info("V4Log", "key-info", "value-info")
		logger.V(5).Info("V5Log", "key-info", "value-info")
		logger.V(8).Info("V8Log", "key-debug", "value-debug")
	})
}
