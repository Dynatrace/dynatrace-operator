package hostvolumes

import (
	"context"
	"testing"

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

		appMount, err := publisher.db.ReadOSMount(context.Background(), metadata.OSMount{VolumeMetaID: testVolumeId})
		require.Error(t, err)
		assert.Nil(t, appMount)
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

	osMount := metadata.OSMount{VolumeMetaID: testVolumeId, VolumeMeta: metadata.VolumeMeta{ID: testVolumeId}, TenantUUID: testTenantUUID}
	err := publisher.db.CreateOSMount(context.Background(), &osMount)
	require.NoError(t, err)
}

func mockDynakube(t *testing.T, publisher *HostVolumePublisher) {
	tenantConfig := metadata.TenantConfig{Name: testDynakubeName, TenantUUID: testTenantUUID, DownloadedCodeModuleVersion: "some-version", MaxFailedMountAttempts: 0}
	err := publisher.db.CreateTenantConfig(context.Background(), &tenantConfig)
	require.NoError(t, err)
}

func mockDynakubeWithoutVersion(t *testing.T, publisher *HostVolumePublisher) {
	tenantConfig := metadata.TenantConfig{Name: testDynakubeName, TenantUUID: testTenantUUID, DownloadedCodeModuleVersion: "", MaxFailedMountAttempts: 0}
	err := publisher.db.CreateTenantConfig(context.Background(), &tenantConfig)
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, publisher *HostVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)

	volume, err := publisher.db.ReadOSMount(context.Background(), metadata.OSMount{VolumeMetaID: testVolumeId})
	require.NoError(t, err)
	assert.Equal(t, testVolumeId, volume.VolumeMetaID)
	assert.Equal(t, testTenantUUID, volume.TenantUUID)
}

func assertReferencesForUnpublishedVolume(t *testing.T, publisher *HostVolumePublisher) {
	volume, err := publisher.db.ReadOSMount(context.Background(), metadata.OSMount{VolumeMetaID: testVolumeId})
	require.Error(t, err)
	assert.Nil(t, volume)
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
