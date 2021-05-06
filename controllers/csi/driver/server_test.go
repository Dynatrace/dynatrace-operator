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
