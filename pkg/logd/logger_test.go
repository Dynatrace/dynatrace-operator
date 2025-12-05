package logd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestDefaultLogLevel(t *testing.T) {
	logLevel := readLogLevelFromEnv()
	assert.Equal(t, zapcore.InfoLevel, logLevel)
}

func TestLogLevelFromEnv(t *testing.T) {
	t.Setenv(LogLevelEnv, "debug")

	logLevel := readLogLevelFromEnv()
	assert.Equal(t, zapcore.DebugLevel, logLevel)
}

func TestLogLevelFromEnvEmptyString(t *testing.T) {
	t.Setenv(LogLevelEnv, "unknown")

	logLevel := readLogLevelFromEnv()
	assert.Equal(t, zapcore.InfoLevel, logLevel)
}

func TestLogger(t *testing.T) {
	t.Run("log level Info", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := CreateLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), zapcore.InfoLevel)

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
	t.Run("log level Debug", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := CreateLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), zapcore.DebugLevel)

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.Contains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
	t.Run("log level default without env", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := CreateLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), readLogLevelFromEnv())

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
}
