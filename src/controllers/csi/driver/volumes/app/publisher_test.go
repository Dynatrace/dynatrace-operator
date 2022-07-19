package appvolumes

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPodUID       = "a-pod"
	testVolumeId     = "a-volume"
	testTargetPath   = "/path/to/container/filesystem/opt/dynatrace/oneagent-paas"
	testTenantUUID   = "a-tenant-uuid"
	testAgentVersion = "1.2-3"
	testDynakubeName = "a-dynakube"
	testImageDigest  = "sha256:123456789"
)

func TestPublishVolume(t *testing.T) {
	t.Run(`using url`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(t, mounter)
		mockUrlDynakubeMetadata(t, &publisher)

		response, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())
		require.NoError(t, err)
		assert.NotNil(t, response)

		require.NotEmpty(t, mounter.MountPoints)

		assert.Equal(t, "overlay", mounter.MountPoints[0].Device)
		assert.Equal(t, "overlay", mounter.MountPoints[0].Type)
		assert.Equal(t, []string{
			"lowerdir=/a-tenant-uuid/bin/1.2-3",
			"upperdir=/a-tenant-uuid/run/a-volume/var",
			"workdir=/a-tenant-uuid/run/a-volume/work"},
			mounter.MountPoints[0].Opts)
		assert.Equal(t, "/a-tenant-uuid/run/a-volume/mapped", mounter.MountPoints[0].Path)

		assert.Equal(t, "overlay", mounter.MountPoints[1].Device)
		assert.Equal(t, "", mounter.MountPoints[1].Type)
		assert.Equal(t, []string{"bind"}, mounter.MountPoints[1].Opts)
		assert.Equal(t, testTargetPath, mounter.MountPoints[1].Path)

		assertReferencesForPublishedVolume(t, &publisher, mounter)
	})

	t.Run(`using code modules image`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(t, mounter)
		mockImageDynakubeMetadata(t, &publisher)

		response, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())
		require.NoError(t, err)
		assert.NotNil(t, response)

		require.NotEmpty(t, mounter.MountPoints)

		assert.Equal(t, "overlay", mounter.MountPoints[0].Device)
		assert.Equal(t, "overlay", mounter.MountPoints[0].Type)
		assert.Equal(t, []string{
			"lowerdir=/a-tenant-uuid/config:/codemodules/" + testImageDigest,
			"upperdir=/a-tenant-uuid/run/a-volume/var",
			"workdir=/a-tenant-uuid/run/a-volume/work"},
			mounter.MountPoints[0].Opts)
		assert.Equal(t, "/a-tenant-uuid/run/a-volume/mapped", mounter.MountPoints[0].Path)

		assert.Equal(t, "overlay", mounter.MountPoints[1].Device)
		assert.Equal(t, "", mounter.MountPoints[1].Type)
		assert.Equal(t, []string{"bind"}, mounter.MountPoints[1].Opts)
		assert.Equal(t, testTargetPath, mounter.MountPoints[1].Path)

		assertReferencesForPublishedVolumeWithCodeModulesImage(t, &publisher, mounter)
	})
}

func TestUnpublishVolume(t *testing.T) {
	t.Run(`valid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", testTenantUUID, testVolumeId)},
		})
		publisher := newPublisherForTesting(t, mounter)
		mockPublishedVolume(t, &publisher)

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &publisher)
	})

	t.Run(`invalid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", testTenantUUID, testVolumeId)},
		})
		publisher := newPublisherForTesting(t, mounter)

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		require.NoError(t, err)
		require.Nil(t, response)
		require.NotEmpty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &publisher)
	})
}

func TestNodePublishAndUnpublishVolume(t *testing.T) {
	resetMetrics()
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)
	mockUrlDynakubeMetadata(t, &publisher)

	publishResponse, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())

	assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

	assert.NoError(t, err)
	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &publisher, mounter)

	unpublishResponse, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

	assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

	require.NoError(t, err)
	require.NotNil(t, unpublishResponse)
	require.Empty(t, mounter.MountPoints)
	assertNoReferencesForUnpublishedVolume(t, &publisher)
}

func TestStoreAndLoadPodInfo(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)

	bindCfg := &csivolumes.BindConfig{
		Version:    testAgentVersion,
		TenantUUID: testTenantUUID,
	}

	volumeCfg := createTestVolumeConfig()

	err := publisher.storeVolume(bindCfg, volumeCfg)
	require.NoError(t, err)
	volume, err := publisher.loadVolume(volumeCfg.VolumeID)
	require.NoError(t, err)
	require.NotNil(t, volume)
	assert.Equal(t, testVolumeId, volume.VolumeID)
	assert.Equal(t, testPodUID, volume.PodName)
	assert.Equal(t, testAgentVersion, volume.Version)
	assert.Equal(t, testTenantUUID, volume.TenantUUID)
}

func TestLoadPodInfo_Empty(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)

	volume, err := publisher.loadVolume(testVolumeId)
	require.NoError(t, err)
	require.Nil(t, volume)
}

func newPublisherForTesting(t *testing.T, mounter *mount.FakeMounter) AppVolumePublisher {
	objects := []client.Object{
		&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDynakubeName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDynakubeName,
			},
		},
	}

	csiOptions := dtcsi.CsiOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return AppVolumePublisher{
		client:  fake.NewClient(objects...),
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		path:    metadata.PathResolver{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	mockUrlDynakubeMetadata(t, publisher)
	err := publisher.db.InsertVolume(metadata.NewVolume(testVolumeId, testPodUID, testAgentVersion, testTenantUUID))
	require.NoError(t, err)
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Inc()
}

func mockUrlDynakubeMetadata(t *testing.T, publisher *AppVolumePublisher) {
	err := publisher.db.InsertDynakube(metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, ""))
	require.NoError(t, err)
}

func mockImageDynakubeMetadata(t *testing.T, publisher *AppVolumePublisher) {
	err := publisher.db.InsertDynakube(metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, testImageDigest))
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, publisher *AppVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := publisher.loadVolume(testVolumeId)
	require.NoError(t, err)
	assert.Equal(t, volume.VolumeID, testVolumeId)
	assert.Equal(t, volume.PodName, testPodUID)
	assert.Equal(t, volume.Version, testAgentVersion)
	assert.Equal(t, volume.TenantUUID, testTenantUUID)
}

func assertReferencesForPublishedVolumeWithCodeModulesImage(t *testing.T, publisher *AppVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := publisher.loadVolume(testVolumeId)
	require.NoError(t, err)
	assert.Equal(t, volume.VolumeID, testVolumeId)
	assert.Equal(t, volume.PodName, testPodUID)
	assert.Equal(t, volume.Version, testImageDigest)
	assert.Equal(t, volume.TenantUUID, testTenantUUID)
}

func assertNoReferencesForUnpublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	volume, err := publisher.loadVolume(testVolumeId)
	require.NoError(t, err)
	require.Nil(t, volume)
}

func resetMetrics() {
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Set(0)
}

func createTestVolumeConfig() *csivolumes.VolumeConfig {
	return &csivolumes.VolumeConfig{
		VolumeInfo:   *createTestVolumeInfo(),
		PodName:      testPodUID,
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
