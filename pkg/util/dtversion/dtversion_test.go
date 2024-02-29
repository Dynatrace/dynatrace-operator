package dtversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSemver(t *testing.T) {
	type test struct {
		title         string
		inputVersion  string
		outputVersion string
		expectError   bool
	}

	testCases := []test{
		{
			title:         "full version",
			inputVersion:  "1.267.209.20231204-134602",
			outputVersion: "v1.267.209",
			expectError:   false,
		},
		{
			title:         "partial version works - no dash part at the end",
			inputVersion:  "1.277.209.20231204",
			outputVersion: "v1.277.209",
			expectError:   false,
		},
		{
			title:         "partial version works - only till patch version",
			inputVersion:  "1.277.209",
			outputVersion: "v1.277.209",
			expectError:   false,
		},
		{
			title:         "partial version works - only till minor version",
			inputVersion:  "1.277",
			outputVersion: "v1.277.0",
			expectError:   false,
		},
		{
			title:         "partial version works - only till mayor version",
			inputVersion:  "1",
			outputVersion: "v1.0.0",
			expectError:   false,
		},
		{
			title:         "empty version works",
			inputVersion:  "",
			outputVersion: "",
			expectError:   false,
		},
		{
			title:         "works with 'v' prefix",
			inputVersion:  "v1.277",
			outputVersion: "v1.277.0",
			expectError:   false,
		},
		{
			title:         "works without 'v' prefix",
			inputVersion:  "1.275.0",
			outputVersion: "v1.275.0",
			expectError:   false,
		},
		{
			title:         "malformed version - returns error",
			inputVersion:  ".4.malformed-",
			outputVersion: "",
			expectError:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			actual, err := ToSemver(testCase.inputVersion)

			if testCase.expectError {
				require.Error(t, err)
				require.Empty(t, actual)
				assert.Contains(t, err.Error(), testCase.inputVersion)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.outputVersion, actual)
		})
	}
}
