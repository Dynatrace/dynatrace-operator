package csidriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSIDriverServer_parseEndpoint(t *testing.T) {
	t.Run(`valid unix endpoint`, func(t *testing.T) {
		testEndpoint := "unix:///some/socket"
		protocol, address, err := parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "unix", protocol)
		assert.Equal(t, "/some/socket", address)

		testEndpoint = "UNIX:///SOME/socket"
		protocol, address, err = parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "UNIX", protocol)
		assert.Equal(t, "/SOME/socket", address)

		testEndpoint = "uNiX:///SOME/socket://weird-uri"
		protocol, address, err = parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "uNiX", protocol)
		assert.Equal(t, "/SOME/socket://weird-uri", address)
	})
	t.Run(`valid tcp endpoint`, func(t *testing.T) {
		testEndpoint := "tcp://127.0.0.1/some/endpoint"
		protocol, address, err := parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "tcp", protocol)
		assert.Equal(t, "127.0.0.1/some/endpoint", address)

		testEndpoint = "TCP:///localhost/some/ENDPOINT"
		protocol, address, err = parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "TCP", protocol)
		assert.Equal(t, "/localhost/some/ENDPOINT", address)

		testEndpoint = "tCp://localhost/some/ENDPOINT://weird-uri"
		protocol, address, err = parseEndpoint(testEndpoint)

		require.NoError(t, err)
		assert.Equal(t, "tCp", protocol)
		assert.Equal(t, "localhost/some/ENDPOINT://weird-uri", address)
	})
	t.Run(`invalid endpoint`, func(t *testing.T) {
		testEndpoint := "udp://website.com/some/endpoint"
		protocol, address, err := parseEndpoint(testEndpoint)

		require.EqualError(t, err, "invalid endpoint: "+testEndpoint)
		assert.Equal(t, "", protocol)
		assert.Equal(t, "", address)
	})
}
