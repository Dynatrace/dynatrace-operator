package troubleshoot

import (
	"bytes"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runWithTestLogger(function func(log logger.DtLogger)) string {
	logBuffer := bytes.Buffer{}
	logger := NewTroubleshootLoggerToWriter(&logBuffer)
	function(logger)

	return logBuffer.String()
}

func getNullLogger(t *testing.T) logger.DtLogger {
	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)

	return NewTroubleshootLoggerToWriter(devNull)
}

func TestTroubleshootLogger(t *testing.T) {
	const testLogOutput = "test log output"

	logOutput := runWithTestLogger(func(log logger.DtLogger) {
		logInfof(log, testLogOutput)
	})

	assert.Contains(t, logOutput, testLogOutput)
}
