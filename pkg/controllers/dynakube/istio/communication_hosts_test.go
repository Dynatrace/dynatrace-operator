package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommunicationHost(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    CommunicationHost
		expectError bool
	}{
		{
			name:  "https with default port",
			input: "https://example.live.dynatrace.com/communication",
			expected: CommunicationHost{
				Protocol: "https",
				Host:     "example.live.dynatrace.com",
				Port:     443,
			},
		},
		{
			name:  "https with custom port",
			input: "https://managedhost.com:9999/here/communication",
			expected: CommunicationHost{
				Protocol: "https",
				Host:     "managedhost.com",
				Port:     9999,
			},
		},
		{
			name:  "https with IP address and custom port",
			input: "https://10.0.0.1:8000/communication",
			expected: CommunicationHost{
				Protocol: "https",
				Host:     "10.0.0.1",
				Port:     8000,
			},
		},
		{
			name:  "http with default port",
			input: "http://insecurehost/communication",
			expected: CommunicationHost{
				Protocol: "http",
				Host:     "insecurehost",
				Port:     80,
			},
		},
		{
			name:        "invalid port",
			input:       "https://managedhost.com:notaport/here/communication",
			expectError: true,
		},
		{
			name:        "missing protocol",
			input:       "example.live.dynatrace.com:80/communication",
			expectError: true,
		},
		{
			name:        "unsupported protocol ftp",
			input:       "ftp://randomhost.com:80/communication",
			expectError: true,
		},
		{
			name:        "unsupported protocol unix",
			input:       "unix:///some/local/file",
			expectError: true,
		},
		{
			name:        "unparseable input",
			input:       "://::::",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ch, err := NewCommunicationHost(tc.input)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, ch)
			}
		})
	}
}

func TestNewCommunicationHosts(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    []CommunicationHost
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []CommunicationHost{},
		},
		{
			name:  "single endpoint",
			input: "https://example.live.dynatrace.com/communication",
			expected: []CommunicationHost{
				{
					Protocol: "https",
					Host:     "example.live.dynatrace.com",
					Port:     443,
				},
			},
		},
		{
			name:  "multiple endpoints",
			input: "https://example.live.dynatrace.com/communication,https://managedhost.com:9999/here/communication",
			expected: []CommunicationHost{
				{
					Protocol: "https",
					Host:     "example.live.dynatrace.com",
					Port:     443,
				},
				{
					Protocol: "https",
					Host:     "managedhost.com",
					Port:     9999,
				},
			},
		},
		{
			name:  "duplicate endpoints are deduplicated",
			input: "https://example.live.dynatrace.com/communication,https://example.live.dynatrace.com/communication",
			expected: []CommunicationHost{
				{
					Protocol: "https",
					Host:     "example.live.dynatrace.com",
					Port:     443,
				},
			},
		},
		{
			name:  "mixed protocols",
			input: "https://secure.example.com/communication,http://insecure.example.com/communication",
			expected: []CommunicationHost{
				{
					Protocol: "http",
					Host:     "insecure.example.com",
					Port:     80,
				},
				{
					Protocol: "https",
					Host:     "secure.example.com",
					Port:     443,
				},
			},
		},
		{
			name:        "invalid endpoint in list",
			input:       "https://valid.com/communication,invalidendpoint",
			expectError: true,
		},
		{
			name:        "invalid port in list",
			input:       "https://valid.com/communication,https://invalid.com:notaport/communication",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hosts, err := NewCommunicationHosts(tc.input)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, hosts)
			}
		})
	}
}
