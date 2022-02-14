package csidriver

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/mount"
)

const (
	testTargetNotExist   = "not-exists"
	testTargetError      = "error"
	testTargetNotMounted = "not-mounted"
	testTargetMounted    = "mounted"

	testError = "test error message"
)

type fakeMounter struct {
	mount.FakeMounter
}

func (*fakeMounter) IsLikelyNotMountPoint(target string) (bool, error) {
	if target == testTargetNotExist {
		return false, os.ErrNotExist
	} else if target == testTargetError {
		return false, fmt.Errorf(testError)
	} else if target == testTargetMounted {
		return true, nil
	}
	return false, nil
}

func TestCSIDriverServer_IsMounted(t *testing.T) {
	t.Run(`mount point does not exist`, func(t *testing.T) {
		mounted, err := isMounted(&fakeMounter{}, testTargetNotExist)
		assert.NoError(t, err)
		assert.False(t, mounted)
	})
	t.Run(`mounter throws error`, func(t *testing.T) {
		mounted, err := isMounted(&fakeMounter{}, testTargetError)

		assert.EqualError(t, err, "rpc error: code = Internal desc = test error message")
		assert.False(t, mounted)
	})
	t.Run(`mount point is not mounted`, func(t *testing.T) {
		mounted, err := isMounted(&fakeMounter{}, testTargetNotMounted)

		assert.NoError(t, err)
		assert.True(t, mounted)
	})
	t.Run(`mount point is mounted`, func(t *testing.T) {
		mounted, err := isMounted(&fakeMounter{}, testTargetMounted)

		assert.NoError(t, err)
		assert.False(t, mounted)
	})
}

func TestCSIDriverServer_parseEndpoint(t *testing.T) {
	t.Run(`valid unix endpoint`, func(t *testing.T) {
		testEndpoint := "unix:///some/socket"
		protocol, address, err := parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "unix", protocol)
		assert.Equal(t, "/some/socket", address)

		testEndpoint = "UNIX:///SOME/socket"
		protocol, address, err = parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "UNIX", protocol)
		assert.Equal(t, "/SOME/socket", address)

		testEndpoint = "uNiX:///SOME/socket://weird-uri"
		protocol, address, err = parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "uNiX", protocol)
		assert.Equal(t, "/SOME/socket://weird-uri", address)
	})
	t.Run(`valid tcp endpoint`, func(t *testing.T) {
		testEndpoint := "tcp://127.0.0.1/some/endpoint"
		protocol, address, err := parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "tcp", protocol)
		assert.Equal(t, "127.0.0.1/some/endpoint", address)

		testEndpoint = "TCP:///localhost/some/ENDPOINT"
		protocol, address, err = parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "TCP", protocol)
		assert.Equal(t, "/localhost/some/ENDPOINT", address)

		testEndpoint = "tCp://localhost/some/ENDPOINT://weird-uri"
		protocol, address, err = parseEndpoint(testEndpoint)

		assert.NoError(t, err)
		assert.Equal(t, "tCp", protocol)
		assert.Equal(t, "localhost/some/ENDPOINT://weird-uri", address)
	})
	t.Run(`invalid endpoint`, func(t *testing.T) {
		testEndpoint := "udp://website.com/some/endpoint"
		protocol, address, err := parseEndpoint(testEndpoint)

		assert.EqualError(t, err, "invalid endpoint: "+testEndpoint)
		assert.Equal(t, "", protocol)
		assert.Equal(t, "", address)
	})
}
