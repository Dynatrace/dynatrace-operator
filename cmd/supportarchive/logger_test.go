package supportarchive

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportArchiveLogger(t *testing.T) {
	logBuffer := bytes.Buffer{}
	logger := newSupportArchiveLogger(&logBuffer)

	logger.Info("info message")
	logger.Error(assert.AnError, "error message")

	logLines := logBuffer.String()

	assert.Contains(t, logLines, "info message")
	assert.Contains(t, logLines, "error message")
	assert.Contains(t, logLines, supportArchiveLoggerName)
}
