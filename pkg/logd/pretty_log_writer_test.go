package logd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorPrettify_Write(t *testing.T) {
	t.Run(`Write unescapes newlines and tabs`, func(t *testing.T) {
		// backticks ` interpret newlines and tabs as their escaped version
		// I.e., `\n` = "\\n"
		// Which is why backslashes remain in the result text
		testString := `{"newlines" : "some\\n text to \n\\n escape newlines", "tabs": "some\t text\t to \\t\t escape tabs", "mixed": "some\n\\n text \n\t to escape \\n\\t tabs \t\\n\n\\t"}`
		expectedString := "{\"mixed\":\"some\n\\\n text \n\t to escape \\\n\\\t tabs \t\\\n\n\\\t\",\"newlines\":\"some\\\n text to \n\\\n escape newlines\",\"tabs\":\"some\t text\t to \\\t\t escape tabs\"}\n"

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Greater(t, written, 0)
		assert.Equal(t, expectedString, bufferString.String())
	})
	t.Run(`Write replaces "stacktrace" with "errorVerbose"`, func(t *testing.T) {
		testString := `{"stacktrace":"stacktrace","errorVerbose":"errorVerbose"}`
		expectedString := "{\"stacktrace\":\"errorVerbose\"}\n"

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Greater(t, written, 0)
		assert.Equal(t, expectedString, bufferString.String())
	})
	t.Run("Write writes non json message to output", func(t *testing.T) {
		testString := "this is a normal message"
		expectedString := "this is a normal message"

		bufferString := bytes.NewBufferString("")
		errPrettify := NewPrettyLogWriter(WithWriter(bufferString))
		written, err := errPrettify.Write([]byte(testString))

		require.NoError(t, err)
		assert.Greater(t, written, 0)
		assert.Equal(t, expectedString, bufferString.String())
	})
}
