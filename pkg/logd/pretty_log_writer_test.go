package logd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorPrettify_Write(t *testing.T) {
	t.Run("Don't touch input without stacktrace, to keep the order intact", func(t *testing.T) {
		testString := `{"level":"info","ts":"2024-05-28T14:00:03.031Z","logger":"csi-driver","msg":"starting listener","protocol":"unix","address":"csi/csi.sock"}`

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Positive(t, written)
		assert.Equal(t, testString, bufferString.String())
	})
	t.Run("Write replaces 'stacktrace' with 'errorVerbose'", func(t *testing.T) {
		testString := `{"stacktrace":"stacktrace","errorVerbose":"errorVerbose"}`
		expectedString := "{\"stacktrace\":\"errorVerbose\"}\n"

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Positive(t, written)
		assert.Equal(t, expectedString, bufferString.String())
	})
	t.Run("Write writes non json message to output", func(t *testing.T) {
		testString := "this is a normal message"
		expectedString := "this is a normal message"

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Positive(t, written)
		assert.Equal(t, expectedString, bufferString.String())
	})
}
