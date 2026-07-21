// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtapiurl

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsThirdGen(t *testing.T) {
	assert.True(t, isThirdGen("tenant.apps.dynatrace.com"))
	assert.True(t, isThirdGen("tenant.dev.apps.dynatracelabs.com"))
	assert.False(t, isThirdGen("tenant.live.dynatrace.com"))
	assert.False(t, isThirdGen("tenant.dynatrace.com"))
}

func TestMapToSecondGen(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://tenant.apps.dynatrace.com",
			expected: "https://tenant.live.dynatrace.com/api",
		},
		{
			input:    "https://tenant.sprint.apps.dynatrace.com",
			expected: "https://tenant.sprint.dynatrace.com/api",
		},
		{
			input:    "https://tenant.dev.apps.dynatrace.com",
			expected: "https://tenant.dev.dynatrace.com/api",
		},
		{
			input:    "https://tenant.apps.dynatrace.com:8443",
			expected: "https://tenant.live.dynatrace.com:8443/api",
		},
		{
			input:    "https://tenant.sprint.apps.dynatrace.com:9090",
			expected: "https://tenant.sprint.dynatrace.com:9090/api",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			u, err := url.Parse(tc.input)
			require.NoError(t, err)

			mapToSecondGen(u)
			assert.Equal(t, tc.expected, u.String())
		})
	}
}

func TestMapToSecondGenLeavesSecondGenUntouched(t *testing.T) {
	input := "https://tenant.live.dynatrace.com/api"

	u, err := url.Parse(input)
	require.NoError(t, err)

	mapToSecondGen(u)
	assert.Equal(t, input, u.String())
}

func TestToSecondGen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "3rd gen url is remapped to 2nd gen",
			input:    "https://tenant.apps.dynatrace.com",
			expected: "https://tenant.live.dynatrace.com/api",
		},
		{
			name:     "3rd gen url with subdomain is remapped to 2nd gen",
			input:    "https://tenant.sprint.apps.dynatrace.com",
			expected: "https://tenant.sprint.dynatrace.com/api",
		},
		{
			name:     "2nd gen url is returned unchanged",
			input:    "https://tenant.live.dynatrace.com/api",
			expected: "https://tenant.live.dynatrace.com/api",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ToSecondGen(tc.input))
		})
	}
}
