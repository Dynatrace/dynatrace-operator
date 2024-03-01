package hostvolumes

import (
	"context"
	"testing"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/mount"
)

const (
	testVolumeId     = "a-volume"
	testTargetPath   = "/path/to/container/filesystem"
	testTenantUUID   = "a-tenant-uuid"
	testDynakubeName = "a-dynakube"
)

func TestPublishVolume(t *testing.T) {
	t.Run(`ready dynakube`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)

		mockDynakube(t, &publisher)

		response, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)
		assertReferencesForPublishedVolume(t, &publisher, mounter)
	})
	t.Run(`not ready dynakube`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)

		mockDynakubeWithoutVersion(t, &publisher)

		response, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)
		assertReferencesForPublishedVolume(t, &publisher, mounter)
	})
}

func TestUnpublishVolume(t *testing.T) {
	t.Run(`valid metadata`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
		})
		publisher := newPublisherForTesting(mounter)
		mockPublishedvolume(t, &publisher)

		response, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Empty(t, mounter.MountPoints)
		assertReferencesForUnpublishedVolume(t, &publisher)
	})

	t.Run(`invalid metadata`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
		})
		publisher := newPublisherForTesting(mounter)

		response, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)

		volume, err := publisher.db.GetOsAgentVolumeViaVolumeID(context.Background(), testVolumeId)
		require.NoError(t, err)
		assert.Nil(t, volume)
	})
}

func TestNodePublishAndUnpublishVolume(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
	mockDynakube(t, &publisher)

	publishResponse, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())
	require.NoError(t, err)

	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &publisher, mounter)

	unpublishResponse, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

	require.NoError(t, err)
	assert.NotNil(t, unpublishResponse)
	assert.Empty(t, mounter.MountPoints)
	assertReferencesForUnpublishedVolume(t, &publisher)
}

func newPublisherForTesting(mounter *mount.FakeMounter) HostVolumePublisher {
	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return HostVolumePublisher{
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		path:    metadata.PathResolver{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedvolume(t *testing.T, publisher *HostVolumePublisher) {
	mockDynakube(t, publisher)

	now := time.Now()
	err := publisher.db.InsertOsAgentVolume(context.Background(), metadata.NewOsAgentVolume(testVolumeId, testTenantUUID, true, &now))
	require.NoError(t, err)
}

func mockDynakube(t *testing.T, publisher *HostVolumePublisher) {
	err := publisher.db.InsertDynakube(context.Background(), metadata.NewDynakube(testDynakubeName, testTenantUUID, "some-version", "", 0))
	require.NoError(t, err)
}

func mockDynakubeWithoutVersion(t *testing.T, publisher *HostVolumePublisher) {
	err := publisher.db.InsertDynakube(context.Background(), metadata.NewDynakube(testDynakubeName, testTenantUUID, "", "", 0))
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, publisher *HostVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)

	volume, err := publisher.db.GetOsAgentVolumeViaVolumeID(context.Background(), testVolumeId)
	require.NoError(t, err)
	assert.Equal(t, testVolumeId, volume.VolumeID)
	assert.Equal(t, testTenantUUID, volume.TenantUUID)
	assert.True(t, volume.Mounted)
}

func assertReferencesForUnpublishedVolume(t *testing.T, publisher *HostVolumePublisher) {
	volume, err := publisher.db.GetOsAgentVolumeViaVolumeID(context.Background(), testVolumeId)
	require.NoError(t, err)
	assert.NotNil(t, volume)
	assert.False(t, volume.Mounted)
}

func createTestVolumeConfig() *csivolumes.VolumeConfig {
	return &csivolumes.VolumeConfig{
		VolumeInfo:   *createTestVolumeInfo(),
		Mode:         Mode,
		DynakubeName: testDynakubeName,
	}
}

func createTestVolumeInfo() *csivolumes.VolumeInfo {
	return &csivolumes.VolumeInfo{
		VolumeID:   testVolumeId,
		TargetPath: testTargetPath,
	}
}
