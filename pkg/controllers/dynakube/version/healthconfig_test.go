package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOneAgentHealthConfig(t *testing.T) {
	type test struct {
		title           string
		inputVersion    string
		expectedCommand []string
		expectError     bool
	}

	testCases := []test{
		{
			title:           "get healthConfig with test as CMD - current versions case",
			inputVersion:    "1.277.209.20231204-134602",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "get healthConfig with test as shell script - old version case",
			inputVersion:    "1.267.209.20231204-134602",
			expectedCommand: preThresholdHealthCheck,
			expectError:     false,
		},
		{
			title:           "partial version works - no dash part at the end",
			inputVersion:    "1.277.209.20231204",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "partial version works - only till patch version",
			inputVersion:    "1.277.209",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "partial version works - only till minor version",
			inputVersion:    "1.277",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "partial version works - only till mayor version",
			inputVersion:    "1",
			expectedCommand: preThresholdHealthCheck,
			expectError:     false,
		},
		{
			title:           "empty version works - default is the current healthcheck",
			inputVersion:    "",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "exact match works",
			inputVersion:    healthCheckVersionThreshold,
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "works with 'v' prefix",
			inputVersion:    "v1.277",
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
		{
			title:           "works without 'v' prefix",
			inputVersion:    "1.275.0",
			expectedCommand: preThresholdHealthCheck,
			expectError:     false,
		},
		{
			title:           "malformed version - returns error",
			inputVersion:    ".4.malformed-",
			expectedCommand: nil,
			expectError:     true,
		},
		{
			title:           "incase of custom-image",
			inputVersion:    string(status.CustomImageVersionSource),
			expectedCommand: currentHealthCheck,
			expectError:     false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			healthConfig, err := getOneAgentHealthConfig(testCase.inputVersion)

			if testCase.expectError {
				require.Error(t, err)
				require.Nil(t, healthConfig)
				assert.Contains(t, err.Error(), testCase.inputVersion)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, healthConfig)
			assert.Equal(t, testCase.expectedCommand, healthConfig.Test)
			assert.Equal(t, defaultHealthConfigInterval, healthConfig.Interval)
			assert.Equal(t, defaultHealthConfigTimeout, healthConfig.Timeout)
			assert.Equal(t, defaultHealthConfigStartPeriod, healthConfig.StartPeriod)
			assert.Equal(t, defaultHealthConfigRetries, healthConfig.Retries)
		})
	}
}
