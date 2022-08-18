package metadata

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testName                   = "test-name"
	testID                     = "test-id"
	testUID                    = "test-uid"
	testUUID                   = "test-uuid"
	testVersion                = "test-version"
	testDigest                 = "test-digest"
	testMaxFailedMountAttempts = 3
	testMountAttempts          = 1
)

func TestNewDynakube(t *testing.T) {
	t.Run("initializes correctly", func(t *testing.T) {
		dynakube := NewDynakube(testName, testUUID, testVersion, testDigest, testMaxFailedMountAttempts)

		assert.Equal(t, testName, dynakube.Name)
		assert.Equal(t, testUUID, dynakube.TenantUUID)
		assert.Equal(t, testVersion, dynakube.LatestVersion)
		assert.Equal(t, testDigest, dynakube.ImageDigest)
		assert.Equal(t, testMaxFailedMountAttempts, dynakube.MaxFailedMountAttempts)
	})
	t.Run("returns nil if name or uuid is empty", func(t *testing.T) {
		dynakube := NewDynakube("", testUUID, testVersion, testDigest, testMaxFailedMountAttempts)

		assert.Nil(t, dynakube)

		dynakube = NewDynakube(testName, "", testVersion, testDigest, testMaxFailedMountAttempts)

		assert.Nil(t, dynakube)
	})
	t.Run("sets default value for mount attempts if less than 0", func(t *testing.T) {
		dynakube := NewDynakube(testName, testUUID, testVersion, testDigest, -1)

		assert.Equal(t, testName, dynakube.Name)
		assert.Equal(t, testUUID, dynakube.TenantUUID)
		assert.Equal(t, testVersion, dynakube.LatestVersion)
		assert.Equal(t, testDigest, dynakube.ImageDigest)
		assert.Equal(t, defaultMaxFailedMountAttempts, dynakube.MaxFailedMountAttempts)
	})
}

func TestNewVolume(t *testing.T) {
	t.Run("initializes correctly", func(t *testing.T) {
		volume := NewVolume(testID, testName, testVersion, testUUID, testMountAttempts)

		assert.Equal(t, testID, volume.VolumeID)
		assert.Equal(t, testName, volume.PodName)
		assert.Equal(t, testVersion, volume.Version)
		assert.Equal(t, testUUID, volume.TenantUUID)
		assert.Equal(t, testMountAttempts, volume.MountAttempts)
	})
	t.Run("returns nil if id, name, version or uuid is unset", func(t *testing.T) {
		volume := NewVolume("", testName, testVersion, testUUID, testMountAttempts)

		assert.Nil(t, volume)

		volume = NewVolume(testID, "", testVersion, testUUID, testMountAttempts)

		assert.Nil(t, volume)

		volume = NewVolume(testID, testName, "", testUUID, testMountAttempts)

		assert.Nil(t, volume)

		volume = NewVolume(testID, testName, testVersion, "", testMountAttempts)

		assert.Nil(t, volume)

		volume = NewVolume(testID, testName, testVersion, testUUID, 0)

		assert.NotNil(t, volume)
		assert.Equal(t, 0, volume.MountAttempts)
	})
	t.Run("sets default value for mount attempts if less than 0", func(t *testing.T) {
		volume := NewVolume(testID, testName, testVersion, testUUID, -1)

		assert.NotNil(t, volume)
		assert.Equal(t, 0, volume.MountAttempts)

		volume = NewVolume(testID, testName, testVersion, testUUID, -2)

		assert.NotNil(t, volume)
		assert.Equal(t, 0, volume.MountAttempts)
	})
}
