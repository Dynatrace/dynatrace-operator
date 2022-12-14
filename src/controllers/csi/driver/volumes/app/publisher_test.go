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
		publisher := newPublisherForTesting(mounter)
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
		publisher := newPublisherForTesting(mounter)
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

	t.Run(`too many mount attempts`, func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)
		mockFailedPublishedVolume(t, &publisher)

		response, err := publisher.PublishVolume(context.TODO(), createTestVolumeConfig())
		require.NoError(t, err)
		assert.NotNil(t, response)

		require.Empty(t, mounter.MountPoints)
	})
}

func TestHasTooManyMountAttempts(t *testing.T) {
	t.Run(`initial try`, func(t *testing.T) {
		publisher := newPublisherForTesting(nil)
		bindCfg := &csivolumes.BindConfig{
			TenantUUID:       testTenantUUID,
			MaxMountAttempts: dynatracev1beta1.DefaultMaxFailedCsiMountAttempts,
		}
		volumeCfg := createTestVolumeConfig()

		hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(context.TODO(), bindCfg, volumeCfg)

		require.NoError(t, err)
		assert.False(t, hasTooManyAttempts)
		volume, err := publisher.db.GetVolume(context.TODO(), volumeCfg.VolumeID)
		require.NoError(t, err)
		require.NotNil(t, volume)
		assert.Equal(t, 1, volume.MountAttempts)
	})
	t.Run(`too many mount attempts`, func(t *testing.T) {
		publisher := newPublisherForTesting(nil)
		mockFailedPublishedVolume(t, &publisher)
		bindCfg := &csivolumes.BindConfig{
			MaxMountAttempts: dynatracev1beta1.DefaultMaxFailedCsiMountAttempts,
		}

		hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(context.TODO(), bindCfg, createTestVolumeConfig())

		require.NoError(t, err)
		assert.True(t, hasTooManyAttempts)
	})
}

func TestUnpublishVolume(t *testing.T) {
	t.Run(`valid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", testTenantUUID, testVolumeId)},
		})
		publisher := newPublisherForTesting(mounter)
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
		publisher := newPublisherForTesting(mounter)

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		require.NoError(t, err)
		require.Nil(t, response)
		require.NotEmpty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &publisher)
	})

	t.Run(`remove dummy volume created after too many failed attempts`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)
		mockFailedPublishedVolume(t, &publisher)

		response, err := publisher.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, mounter.MountPoints)
	})
}

func TestNodePublishAndUnpublishVolume(t *testing.T) {
	resetMetrics()
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
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
	publisher := newPublisherForTesting(mounter)

	bindCfg := &csivolumes.BindConfig{
		Version:    testAgentVersion,
		TenantUUID: testTenantUUID,
	}

	volumeCfg := createTestVolumeConfig()

	err := publisher.storeVolume(context.TODO(), bindCfg, volumeCfg)
	require.NoError(t, err)
	volume, err := publisher.loadVolume(context.TODO(), volumeCfg.VolumeID)
	require.NoError(t, err)
	require.NotNil(t, volume)
	assert.Equal(t, testVolumeId, volume.VolumeID)
	assert.Equal(t, testPodUID, volume.PodName)
	assert.Equal(t, testAgentVersion, volume.Version)
	assert.Equal(t, testTenantUUID, volume.TenantUUID)
}

func TestLoadPodInfo_Empty(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)

	volume, err := publisher.loadVolume(context.TODO(), testVolumeId)
	require.NoError(t, err)
	require.Nil(t, volume)
}

func TestMountIfDBHasError(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
	publisher.db = &metadata.FakeFailDB{}

	bindCfg := &csivolumes.BindConfig{
		TenantUUID:       testTenantUUID,
		MaxMountAttempts: dynatracev1beta1.DefaultMaxFailedCsiMountAttempts,
	}

	err := publisher.ensureMountSteps(context.TODO(), bindCfg, createTestVolumeConfig())
	require.Error(t, err)
	require.Empty(t, mounter.MountPoints)
}

func newPublisherForTesting(mounter *mount.FakeMounter) AppVolumePublisher {
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

	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

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
	err := publisher.db.InsertVolume(context.TODO(), metadata.NewVolume(testVolumeId, testPodUID, testAgentVersion, testTenantUUID, 0))
	require.NoError(t, err)
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Inc()
}

func mockFailedPublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	mockUrlDynakubeMetadata(t, publisher)
	err := publisher.db.InsertVolume(context.TODO(), metadata.NewVolume(testVolumeId, testPodUID, testAgentVersion, testTenantUUID, dynatracev1beta1.DefaultMaxFailedCsiMountAttempts+1))
	require.NoError(t, err)
}

func mockUrlDynakubeMetadata(t *testing.T, publisher *AppVolumePublisher) {
	err := publisher.db.InsertDynakube(context.TODO(), metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, "", 0))
	require.NoError(t, err)
}

func mockImageDynakubeMetadata(t *testing.T, publisher *AppVolumePublisher) {
	err := publisher.db.InsertDynakube(context.TODO(), metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, testImageDigest, dynatracev1beta1.DefaultMaxFailedCsiMountAttempts))
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, publisher *AppVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := publisher.loadVolume(context.TODO(), testVolumeId)
	require.NoError(t, err)
	assert.Equal(t, volume.VolumeID, testVolumeId)
	assert.Equal(t, volume.PodName, testPodUID)
	assert.Equal(t, volume.Version, testAgentVersion)
	assert.Equal(t, volume.TenantUUID, testTenantUUID)
}

func assertReferencesForPublishedVolumeWithCodeModulesImage(t *testing.T, publisher *AppVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := publisher.loadVolume(context.TODO(), testVolumeId)
	require.NoError(t, err)
	assert.Equal(t, volume.VolumeID, testVolumeId)
	assert.Equal(t, volume.PodName, testPodUID)
	assert.Equal(t, volume.Version, testImageDigest)
	assert.Equal(t, volume.TenantUUID, testTenantUUID)
}

func assertNoReferencesForUnpublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	volume, err := publisher.loadVolume(context.TODO(), testVolumeId)
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
