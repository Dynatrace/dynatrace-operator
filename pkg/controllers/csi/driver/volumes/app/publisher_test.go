package appvolumes

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"k8s.io/utils/mount"
)

const (
	testPodUID             = "a-pod"
	testVolumeId           = "a-volume"
	testTargetPath         = "/path/to/container/filesystem/opt/dynatrace/oneagent-paas"
	testTenantUUID         = "a-tenant-uuid"
	testAgentVersion       = "1.2-3"
	testCodeModuleLocation = "/codemodules/" + testAgentVersion
	testDynakubeName       = "a-dynakube"
	testImageDigest        = "sha256:123456789"
)

func TestPublishVolume(t *testing.T) {
	t.Run("using url", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)
		mockDynakubeMetadata(t, &publisher)
		mockSharedRuxitAgentProcConf(t, &publisher)

		response, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())
		require.NoError(t, err)
		assert.NotNil(t, response)

		require.NotEmpty(t, mounter.MountPoints)

		assert.Equal(t, "overlay", mounter.MountPoints[0].Device)
		assert.Equal(t, "overlay", mounter.MountPoints[0].Type)
		assert.Equal(t, []string{
			"lowerdir=" + testCodeModuleLocation,
			"upperdir=/a-tenant-uuid/run/a-volume/var",
			"workdir=/a-tenant-uuid/run/a-volume/work"},
			mounter.MountPoints[0].Opts)
		assert.Equal(t, "/a-tenant-uuid/run/a-volume/mapped", mounter.MountPoints[0].Path)

		assert.Equal(t, "overlay", mounter.MountPoints[1].Device)
		assert.Equal(t, "", mounter.MountPoints[1].Type)
		assert.Equal(t, []string{"bind"}, mounter.MountPoints[1].Opts)
		assert.Equal(t, testTargetPath, mounter.MountPoints[1].Path)

		confCopied, err := publisher.fs.Exists(publisher.path.OverlayVarRuxitAgentProcConf(testTenantUUID, testVolumeId))
		require.NoError(t, err)
		assert.True(t, confCopied)

		assertReferencesForPublishedVolume(t, &publisher, mounter)
	})

	t.Run("too many mount attempts", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)
		mockFailedPublishedVolume(t, &publisher)

		response, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())
		require.NoError(t, err)
		assert.NotNil(t, response)

		require.Empty(t, mounter.MountPoints)
	})
}

func TestPrepareUpperDir(t *testing.T) {
	testFileContent := []byte{'t', 'e', 's', 't'}

	t.Run("happy path -> file copied from shared dir to overlay dir", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})

		publisher := newPublisherForTesting(mounter)
		mockSharedRuxitAgentProcConf(t, &publisher, testFileContent...)
		testTenantConfig := &metadata.TenantConfig{
			TenantUUID:    testTenantUUID,
			ConfigDirPath: publisher.path.AgentConfigDir(testTenantUUID, testDynakubeName),
		}

		upperDir, err := publisher.prepareUpperDir(testTenantConfig, createTestVolumeConfig())
		require.NoError(t, err)
		require.NotEmpty(t, upperDir)
		assertUpperDirContent(t, &publisher, testFileContent)
	})

	t.Run("sad path -> source file doesn't exist -> error", func(t *testing.T) {
		mounter := mount.NewFakeMounter([]mount.MountPoint{})

		publisher := newPublisherForTesting(mounter)
		testTenantConfig := &metadata.TenantConfig{
			TenantUUID:    testTenantUUID,
			ConfigDirPath: "beep-boop",
		}

		upperDir, err := publisher.prepareUpperDir(testTenantConfig, createTestVolumeConfig())
		require.Error(t, err)
		require.Empty(t, upperDir)

		confCopied, err := publisher.fs.Exists(publisher.path.OverlayVarRuxitAgentProcConf(testTenantUUID, testVolumeId))
		require.NoError(t, err)
		assert.False(t, confCopied)
	})
}

func assertUpperDirContent(t *testing.T, publisher *AppVolumePublisher, expected []byte) {
	content, err := publisher.fs.ReadFile(publisher.path.OverlayVarRuxitAgentProcConf(testTenantUUID, testVolumeId))
	require.NoError(t, err)
	assert.Equal(t, expected, content)
}

func TestHasTooManyMountAttempts(t *testing.T) {
	t.Run(`initial try`, func(t *testing.T) {
		publisher := newPublisherForTesting(nil)
		mockDynakubeMetadata(t, &publisher)

		tenantConfig := &metadata.TenantConfig{
			DownloadedCodeModuleVersion: testAgentVersion,
			TenantUUID:                  testTenantUUID,
			MaxFailedMountAttempts:      dynakube.DefaultMaxFailedCsiMountAttempts,
		}
		volumeCfg := createTestVolumeConfig()

		hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(tenantConfig, volumeCfg)

		require.NoError(t, err)
		assert.False(t, hasTooManyAttempts)

		appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: volumeCfg.VolumeID})
		require.NoError(t, err)
		require.NotNil(t, appMount)
		assert.Equal(t, int64(1), appMount.MountAttempts)
	})

	t.Run(`too many mount attempts`, func(t *testing.T) {
		publisher := newPublisherForTesting(nil)
		mockFailedPublishedVolume(t, &publisher)

		tenantConfig := &metadata.TenantConfig{
			MaxFailedMountAttempts: dynakube.DefaultMaxFailedCsiMountAttempts,
		}

		hasTooManyAttempts, err := publisher.hasTooManyMountAttempts(tenantConfig, createTestVolumeConfig())

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
		assert.InEpsilon(t, 1, testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)), 0.01)

		response, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())
		require.NoError(t, err)

		assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
		assert.InDelta(t, 0, testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)), 0.01)

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

		response, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotEmpty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &publisher)
	})

	t.Run(`remove dummy volume created after too many failed attempts`, func(t *testing.T) {
		resetMetrics()

		mounter := mount.NewFakeMounter([]mount.MountPoint{})
		publisher := newPublisherForTesting(mounter)
		mockFailedPublishedVolume(t, &publisher)

		response, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, mounter.MountPoints)
	})
}

func TestNodePublishAndUnpublishVolume(t *testing.T) {
	resetMetrics()

	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
	mockDynakubeMetadata(t, &publisher)
	mockSharedRuxitAgentProcConf(t, &publisher)

	publishResponse, err := publisher.PublishVolume(context.Background(), createTestVolumeConfig())

	require.NoError(t, err)
	assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
	assert.InEpsilon(t, 1, testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)), 0.01)

	require.NoError(t, err)
	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &publisher, mounter)

	unpublishResponse, err := publisher.UnpublishVolume(context.Background(), createTestVolumeInfo())

	assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
	assert.InDelta(t, 0, testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)), 0.01)

	require.NoError(t, err)
	require.NotNil(t, unpublishResponse)
	require.Empty(t, mounter.MountPoints)
	assertNoReferencesForUnpublishedVolume(t, &publisher)
}

func TestStoreAndLoadPodInfo(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
	mockDynakubeMetadata(t, &publisher)

	tenantConfig := &metadata.TenantConfig{
		DownloadedCodeModuleVersion: testAgentVersion,
		TenantUUID:                  testTenantUUID,
	}

	volumeCfg := createTestVolumeConfig()

	err := publisher.storeVolume(tenantConfig, volumeCfg)
	require.NoError(t, err)
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: volumeCfg.VolumeID})
	require.NoError(t, err)
	require.NotNil(t, appMount)
	assert.Equal(t, testVolumeId, appMount.VolumeMetaID)
	assert.Equal(t, testPodUID, appMount.VolumeMeta.PodName)
	assert.Equal(t, testAgentVersion, appMount.CodeModuleVersion)
}

func TestLoadPodInfo_Empty(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)

	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: testVolumeId})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, appMount)
}

func TestMountIfDBHasError(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(mounter)
	publisher.db = &metadata.FakeFailDB{}

	tenantConfig := &metadata.TenantConfig{
		TenantUUID:             testTenantUUID,
		MaxFailedMountAttempts: dynakube.DefaultMaxFailedCsiMountAttempts,
	}

	err := publisher.ensureMountSteps(tenantConfig, createTestVolumeConfig())
	require.Error(t, err)
	require.Empty(t, mounter.MountPoints)
}

func newPublisherForTesting(mounter *mount.FakeMounter) AppVolumePublisher {
	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return AppVolumePublisher{
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		path:    metadata.PathResolver{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	mockDynakubeMetadata(t, publisher)

	mockAppMount := metadata.AppMount{
		VolumeMeta:        metadata.VolumeMeta{ID: testVolumeId, PodUid: testPodUID},
		CodeModule:        metadata.CodeModule{Version: testAgentVersion},
		VolumeMetaID:      testVolumeId,
		CodeModuleVersion: testAgentVersion,
		MountAttempts:     0,
		Location:          publisher.path.AgentRunDirForVolume(testTenantUUID, testVolumeId),
	}

	err := publisher.db.CreateAppMount(&mockAppMount)
	require.NoError(t, err)
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Inc()
}

func mockFailedPublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	mockDynakubeMetadata(t, publisher)

	appMount := &metadata.AppMount{
		VolumeMeta:        metadata.VolumeMeta{ID: testVolumeId, PodUid: testPodUID},
		CodeModuleVersion: testAgentVersion,
		MountAttempts:     dynakube.DefaultMaxFailedCsiMountAttempts + 1,
		VolumeMetaID:      testVolumeId,
	}

	err := publisher.db.CreateAppMount(appMount)
	require.NoError(t, err)
}

func mockDynakubeMetadata(t *testing.T, publisher *AppVolumePublisher) {
	tenantConfig := metadata.TenantConfig{
		Name:                        testDynakubeName,
		TenantUUID:                  testTenantUUID,
		DownloadedCodeModuleVersion: testAgentVersion,
		ConfigDirPath:               publisher.path.AgentConfigDir(testTenantUUID, testDynakubeName),
		MaxFailedMountAttempts:      0,
	}
	err := publisher.db.CreateTenantConfig(&tenantConfig)
	require.NoError(t, err)

	codeModule := metadata.CodeModule{
		Version:  testAgentVersion,
		Location: testCodeModuleLocation,
	}

	err = publisher.db.CreateCodeModule(&codeModule)
	require.NoError(t, err)
}

func mockSharedRuxitAgentProcConf(t *testing.T, publisher *AppVolumePublisher, content ...byte) {
	file, err := publisher.fs.Create(publisher.path.AgentSharedRuxitAgentProcConf(testTenantUUID, testDynakubeName))
	defer func() { _ = file.Close() }()
	require.NoError(t, err)

	if len(content) > 0 {
		_, err = file.Write(content)
		require.NoError(t, err)
	}
}

func assertReferencesForPublishedVolume(t *testing.T, publisher *AppVolumePublisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)

	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: testVolumeId})
	require.NoError(t, err)
	assert.Equal(t, testVolumeId, appMount.VolumeMetaID)
	assert.Equal(t, testPodUID, appMount.VolumeMeta.PodName)
	assert.Equal(t, testAgentVersion, appMount.CodeModuleVersion)
}

func assertNoReferencesForUnpublishedVolume(t *testing.T, publisher *AppVolumePublisher) {
	appMount, err := publisher.db.ReadAppMount(metadata.AppMount{VolumeMetaID: testVolumeId})
	require.Error(t, err)
	require.Nil(t, appMount)
}

func resetMetrics() {
	agentsVersionsMetric.DeleteLabelValues(testAgentVersion)
	agentsVersionsMetric.DeleteLabelValues(testImageDigest)
}

func createTestVolumeConfig() csivolumes.VolumeConfig {
	return csivolumes.VolumeConfig{
		VolumeInfo:   createTestVolumeInfo(),
		PodName:      testPodUID,
		Mode:         Mode,
		DynakubeName: testDynakubeName,
	}
}

func createTestVolumeInfo() csivolumes.VolumeInfo {
	return csivolumes.VolumeInfo{
		VolumeID:   testVolumeId,
		TargetPath: testTargetPath,
	}
}
