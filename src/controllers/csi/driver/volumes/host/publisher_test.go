package hostvolumes

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testVolumeId     = "a-volume"
	testTargetPath   = "/path/to/container/filesystem"
	testTenantUUID   = "a-tenant-uuid"
	testDynakubeName = "a-dynakube"
)

func TestPublishVolume(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)

	mockDynakube(t, &publisher)

	response, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedStorage(t, &publisher, mounter)
}

func TestUnpublishVolume(t *testing.T) {
	t.Run(`valid metadata`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
		})
		publisher := newPublisherForTesting(t, mounter)
		mockPublishedStorage(t, &publisher)

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Empty(t, mounter.MountPoints)
		assertReferencesForUnpublishedStorage(t, &publisher)
	})

	t.Run(`invalid metadata`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
		})
		publisher := newPublisherForTesting(t, mounter)

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.NoError(t, err)
		assert.Nil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)
		storage, err := publisher.db.GetStorageViaVolumeId(testVolumeId)
		assert.NoError(t, err)
		assert.Nil(t, storage)
	})
}

func TestNodePublishAndUnpublishVolume(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)
	mockDynakube(t, &publisher)

	publishResponse, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())
	assert.NoError(t, err)

	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedStorage(t, &publisher, mounter)

	unpublishResponse, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

	assert.NoError(t, err)
	assert.NotNil(t, unpublishResponse)
	assert.Empty(t, mounter.MountPoints)
	assertReferencesForUnpublishedStorage(t, &publisher)
}

func newPublisherForTesting(t *testing.T, mounter *mount.FakeMounter) HostVolumePublisher {
	objects := []client.Object{
		&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDynakubeName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDynakubeName,
			},
		},
	}

	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return HostVolumePublisher{
		client:  fake.NewClient(objects...),
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		path:    metadata.PathResolver{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedStorage(t *testing.T, publisher *HostVolumePublisher) {
	mockDynakube(t, publisher)
	now := time.Now()
	err := publisher.db.InsertStorage(metadata.NewStorage(testVolumeId, testTenantUUID, true, &now))
	require.NoError(t, err)
}

func mockDynakube(t *testing.T, publisher *HostVolumePublisher) {
	err := publisher.db.InsertDynakube(metadata.NewDynakube(testDynakubeName, testTenantUUID, "doesnt-matter"))
	require.NoError(t, err)
}

func assertReferencesForPublishedStorage(t *testing.T, publisher *HostVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	storage, err := publisher.db.GetStorageViaVolumeId(testVolumeId)
	assert.NoError(t, err)
	assert.Equal(t, storage.VolumeID, testVolumeId)
	assert.Equal(t, storage.TenantUUID, testTenantUUID)
	assert.True(t, storage.Mounted)
}

func assertReferencesForUnpublishedStorage(t *testing.T, publisher *HostVolumePublisher) {
	storage, err := publisher.db.GetStorageViaVolumeId(testVolumeId)
	assert.NoError(t, err)
	assert.NotNil(t, storage)
	assert.False(t, storage.Mounted)
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
