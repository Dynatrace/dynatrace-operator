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
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
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
	testPodUID     = "a-pod"
	testVolumeId   = "a-volume"
	testTargetPath = "/path/to/container/filesystem/opt/dynatrace/oneagent-paas"
)

func TestPublishVolume(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newPublisherForTesting(t, mounter)

	mockOneAgent(t, &server)

	response, err := server.PublishVolume(context.TODO(), createTestVolumeConfig())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &server, mounter)
}

func TestUnpublishVolume(t *testing.T) {
	t.Run(`valid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", testTenantUUID, testVolumeId)},
		})
		server := newPublisherForTesting(t, mounter)
		mockPublishedVolume(t, &server)

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		response, err := server.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Empty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &server)
	})

	t.Run(`invalid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", testTenantUUID, testVolumeId)},
		})
		server := newPublisherForTesting(t, mounter)

		response, err := server.UnpublishVolume(context.TODO(), createTestVolumeInfo())

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &server)
	})
}

func TestCSIDriverServer_NodePublishAndUnpublishVolume(t *testing.T) {
	resetMetrics()
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newPublisherForTesting(t, mounter)
	mockOneAgent(t, &server)

	publishResponse, err := server.PublishVolume(context.TODO(), createTestVolumeConfig())

	assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

	assert.NoError(t, err)
	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &server, mounter)

	unpublishResponse, err := server.UnpublishVolume(context.TODO(), createTestVolumeInfo())

	assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(testAgentVersion)))

	assert.NoError(t, err)
	assert.NotNil(t, unpublishResponse)
	assert.Empty(t, mounter.MountPoints)
	assertNoReferencesForUnpublishedVolume(t, &server)
}

func TestStoreAndLoadPodInfo(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	publisher := newPublisherForTesting(t, mounter)

	bindCfg := &bindConfig{
		version:    testAgentVersion,
		tenantUUID: testTenantUUID,
	}

	volumeCfg := createTestVolumeConfig()

	err := publisher.storeVolume(bindCfg, volumeCfg)
	assert.NoError(t, err)
	volume, err := publisher.loadVolume(volumeCfg.VolumeId)
	assert.NoError(t, err)
	assert.NotNil(t, volume)
	assert.Equal(t, testVolumeId, volume.VolumeID)
	assert.Equal(t, testPodUID, volume.PodName)
	assert.Equal(t, testAgentVersion, volume.Version)
	assert.Equal(t, testTenantUUID, volume.TenantUUID)
}

func TestLoadPodInfo_Empty(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newPublisherForTesting(t, mounter)

	volume, err := server.loadVolume(testVolumeId)
	assert.NoError(t, err)
	assert.NotNil(t, volume)
	assert.Equal(t, &metadata.Volume{}, volume)
}

func newPublisherForTesting(t *testing.T, mounter *mount.FakeMounter) Publisher {
	objects := []client.Object{
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespace,
				Labels: map[string]string{webhook.LabelInstance: testDkName},
			},
		},
		&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDkName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: testDkName,
			},
		},
	}

	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return Publisher{
		client:  fake.NewClient(objects...),
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		path:    metadata.PathResolver{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedVolume(t *testing.T, server *Publisher) {
	mockOneAgent(t, server)
	err := server.db.InsertVolume(metadata.NewVolume(testVolumeId, testPodUID, testAgentVersion, testTenantUUID))
	require.NoError(t, err)
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Inc()
}

func mockOneAgent(t *testing.T, server *Publisher) {
	err := server.db.InsertDynakube(metadata.NewDynakube(testDkName, testTenantUUID, testAgentVersion))
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, server *Publisher, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := server.loadVolume(testVolumeId)
	assert.NoError(t, err)
	assert.Equal(t, volume.VolumeID, testVolumeId)
	assert.Equal(t, volume.PodName, testPodUID)
	assert.Equal(t, volume.Version, testAgentVersion)
	assert.Equal(t, volume.TenantUUID, testTenantUUID)
}

func assertNoReferencesForUnpublishedVolume(t *testing.T, server *Publisher) {
	volume, err := server.loadVolume(testVolumeId)
	assert.NoError(t, err)
	assert.Equal(t, &metadata.Volume{}, volume)
}

func resetMetrics() {
	agentsVersionsMetric.WithLabelValues(testAgentVersion).Set(0)
}

func createTestVolumeConfig() *csivolumes.VolumeConfig {
	return &csivolumes.VolumeConfig{
		VolumeInfo: *createTestVolumeInfo(),
		Namespace:  testNamespace,
		PodName:    testPodUID,
	}
}

func createTestVolumeInfo() *csivolumes.VolumeInfo {
	return &csivolumes.VolumeInfo{
		VolumeId:   testVolumeId,
		TargetPath: testTargetPath,
	}
}
