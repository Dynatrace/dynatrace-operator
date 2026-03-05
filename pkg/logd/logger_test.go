package logd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLogLevel(t *testing.T) {
	verbosity := readVerbosityFromEnv()
	assert.Equal(t, 0, verbosity)
}

func TestLogLevelFromEnv(t *testing.T) {
	t.Setenv(LogLevelEnv, "debug")

	verbosity := readVerbosityFromEnv()
	assert.Equal(t, 1, verbosity)
}

func TestLogLevelFromEnvEmptyString(t *testing.T) {
	t.Setenv(LogLevelEnv, "unknown")

	verbosity := readVerbosityFromEnv()
	assert.Equal(t, 0, verbosity)
}
func TestVerbosityLevelMapping(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{"info level", "info", 0},
		{"debug level", "debug", 1},
		{"trace level", "trace", 2},
		{"extended level", "extended", 3},
		{"verbose level", "verbose", 4},
		{"numeric level", "5", 5},
		{"empty defaults to info", "", 0},
		{"invalid defaults to info", "invalid", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(LogLevelEnv, tt.envValue)
			verbosity := readVerbosityFromEnv()
			assert.Equal(t, tt.expected, verbosity)
		})
	}
}

func TestLogger(t *testing.T) {
	t.Run("log level Info (V0)", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := createLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), 0)

		log.Info("Info message")
		log.V(1).Info("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
	})
	t.Run("log level Debug (V1)", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := createLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), 1)

		log.Info("Info message")
		log.V(1).Info("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.Contains(t, logBuffer.String(), "Debug message")
	})
	t.Run("log level default without env", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := createLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), readVerbosityFromEnv())

		log.Info("Info message")
		log.V(1).Info("Debug message")

		assert.Contains(t, logBuffer.String(), "Info message")
		assert.NotContains(t, logBuffer.String(), "Debug message")
	})
	t.Run("verbosity levels V2-V4", func(t *testing.T) {
		logBuffer := bytes.Buffer{}
		log := createLogger(NewPrettyLogWriter(WithWriter(&logBuffer)), 3)
		log.V(0).Info("V0 message")
		log.V(1).Info("V1 message")
		log.V(2).Info("V2 message")
		log.V(3).Info("V3 message")
		log.V(4).Info("V4 message") // Should not appear (verbosity set to 3)
		output := logBuffer.String()
		assert.Contains(t, output, "V0 message")
		assert.Contains(t, output, "V1 message")
		assert.Contains(t, output, "V2 message")
		assert.Contains(t, output, "V3 message")
		assert.NotContains(t, output, "V4 message")
	})
	t.Run("verbosity levels V2-V4", func(t *testing.T) {
		log := createLogger(NewPrettyLogWriter(WithWriter(t.Output())), 3)
		log.V(0).Info("V0 message")
		log.V(1).Info("V1 message")
		log.V(2).Info("V2 message")
		log.V(3).Info("V3 message")
		log.V(4).Info("V4 message") // Should not appear (verbosity set to 3)
	})
}
