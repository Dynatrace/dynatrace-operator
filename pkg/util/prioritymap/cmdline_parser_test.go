package prioritymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgument(t *testing.T) {
	tests := []struct {
		input             string
		expectedKey       string
		expectedValue     string
		expectedSeparator string
	}{
		{"set-proxy=$(hubert)", "set-proxy", "$(hubert)", "="},
		{"-set-proxy=$(hubert)", "-set-proxy", "$(hubert)", "="},
		{"--set-proxy=$(hubert)", "--set-proxy", "$(hubert)", "="},
		{"----set-proxy=$(hubert)", "----set-proxy", "$(hubert)", "="},
		{"--set-proxy=", "--set-proxy", "", "="},
		{"--simple-flag", "--simple-flag", "", ""},
		{"--set-host-property=OperatorVersion=J30.1", "--set-host-property", "OperatorVersion=J30.1", "="},
	}

	for _, test := range tests {
		t.Run("check:"+test.input, func(t *testing.T) {
			key, separator, value := ParseCommandLineArgument(test.input)

			assert.Equal(t, test.expectedKey, key)
			assert.Equal(t, test.expectedValue, value)
			assert.Equal(t, test.expectedSeparator, separator)
		})
	}
}
