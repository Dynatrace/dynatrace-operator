package logd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLogLevel(t *testing.T) {
	logLevel, err := readLogLevelFromEnv()
	require.NoError(t, err)
	assert.Equal(t, InfoLevel, logLevel)
}

func TestLogLevelFromEnv(t *testing.T) {
	t.Setenv(LogLevelEnv, "debug")

	logLevel, err := readLogLevelFromEnv()
	require.NoError(t, err)
	assert.Equal(t, InfoLevel, logLevel)
}

func TestLogLevelFromEnvEmptyString(t *testing.T) {
	t.Setenv(LogLevelEnv, "unknown")

	logLevel, err := readLogLevelFromEnv()
	require.Error(t, err)
	assert.Equal(t, InfoLevel, logLevel)
}

func TestLogger(t *testing.T) {
	t.Run("log level Info", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := Logger{newZapLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), InfoLevel)}

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
	t.Run("log level Debug", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := Logger{newZapLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), DebugLevel)}

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.Contains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
	t.Run("log level default without env", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		logLevel, err := readLogLevelFromEnv()
		require.NoError(t, err)

		log := Logger{newZapLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), logLevel)}

		log.Info("Info message")
		log.Debug("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
		assert.NotContains(t, logBuffer.String(), "dpanic")
	})
}
