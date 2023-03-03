package troubleshoot

import (
	"bytes"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func runWithTestLogger(function func(log logr.Logger)) string {
	logBuffer := bytes.Buffer{}
	logger := NewTroubleshootLoggerToWriter(&logBuffer)
	function(logger)
	return logBuffer.String()
}

func getNullLogger(t *testing.T) logr.Logger {
	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)
	return NewTroubleshootLoggerToWriter(devNull)
}
