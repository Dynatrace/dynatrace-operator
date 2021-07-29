package csidriver

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/container-storage-interface/spec/lib/go/csi"
	logr "github.com/go-logr/logr/testing"
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
	testTargetNotExist   = "not-exists"
	testTargetError      = "error"
	testTargetNotMounted = "not-mounted"
	testTargetMounted    = "mounted"
	testTargetPath       = "/path/to/container/filesystem/opt/dynatrace/oneagent-paas"

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

func TestServer_NodePublishVolume(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newServerForTesting(t, mounter)
	nodePublishVolumeRequest := &csi.NodePublishVolumeRequest{
		VolumeId: volumeId,
		VolumeContext: map[string]string{
			podNamespaceContextKey: namespace,
			podNameContextKey:      podUID,
		},
		TargetPath: testTargetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
		},
	}
	mockOneAgent(t, &server)

	response, err := server.NodePublishVolume(context.TODO(), nodePublishVolumeRequest)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &server, mounter)
}

func TestServer_NodeUnpublishVolume(t *testing.T) {
	nodeUnpublishVolumeRequest := &csi.NodeUnpublishVolumeRequest{
		VolumeId:   volumeId,
		TargetPath: testTargetPath,
	}

	t.Run(`valid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", tenantUuid, volumeId)},
		})
		server := newServerForTesting(t, mounter)
		mockPublishedVolume(t, &server)

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(agentVersion)))

		response, err := server.NodeUnpublishVolume(context.TODO(), nodeUnpublishVolumeRequest)

		assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(agentVersion)))

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Empty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &server)
	})

	t.Run(`invalid metadata`, func(t *testing.T) {
		resetMetrics()
		mounter := mount.NewFakeMounter([]mount.MountPoint{
			{Path: testTargetPath},
			{Path: fmt.Sprintf("/%s/run/%s/mapped", tenantUuid, volumeId)},
		})
		server := newServerForTesting(t, mounter)

		response, err := server.NodeUnpublishVolume(context.TODO(), nodeUnpublishVolumeRequest)

		assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(agentVersion)))

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, mounter.MountPoints)
		assertNoReferencesForUnpublishedVolume(t, &server)
	})
}

func TestCSIDriverServer_NodePublishAndUnpublishVolume(t *testing.T) {
	resetMetrics()
	nodePublishVolumeRequest := &csi.NodePublishVolumeRequest{
		VolumeId: volumeId,
		VolumeContext: map[string]string{
			podNamespaceContextKey: namespace,
			podNameContextKey:      podUID,
		},
		TargetPath: testTargetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
		},
	}
	nodeUnpublishVolumeRequest := &csi.NodeUnpublishVolumeRequest{
		VolumeId:   volumeId,
		TargetPath: testTargetPath,
	}
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newServerForTesting(t, mounter)
	mockOneAgent(t, &server)

	publishResponse, err := server.NodePublishVolume(context.TODO(), nodePublishVolumeRequest)

	assert.Equal(t, 1, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(1), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(agentVersion)))

	assert.NoError(t, err)
	assert.NotNil(t, publishResponse)
	assert.NotEmpty(t, mounter.MountPoints)
	assertReferencesForPublishedVolume(t, &server, mounter)

	unpublishResponse, err := server.NodeUnpublishVolume(context.TODO(), nodeUnpublishVolumeRequest)

	assert.Equal(t, 0, testutil.CollectAndCount(agentsVersionsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(agentsVersionsMetric.WithLabelValues(agentVersion)))

	assert.NoError(t, err)
	assert.NotNil(t, unpublishResponse)
	assert.Empty(t, mounter.MountPoints)
	assertNoReferencesForUnpublishedVolume(t, &server)
}

func TestStoreAndLoadPodInfo(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newServerForTesting(t, mounter)

	bindCfg := &bindConfig{
		version:    agentVersion,
		tenantUUID: tenantUuid,
	}

	volumeCfg := volumeConfig{
		volumeId:   volumeId,
		targetPath: targetPath,
		namespace:  namespace,
		podName:    podUID,
	}

	err := server.storeVolumeInfo(bindCfg, &volumeCfg)
	assert.NoError(t, err)
	volume, err := server.loadVolumeInfo(volumeCfg.volumeId)
	assert.NoError(t, err)
	assert.NotNil(t, volume)
	assert.Equal(t, volumeId, volume.VolumeID)
	assert.Equal(t, podUID, volume.PodName)
	assert.Equal(t, agentVersion, volume.Version)
	assert.Equal(t, tenantUuid, volume.TenantUUID)
}

func TestLoadPodInfo_Empty(t *testing.T) {
	mounter := mount.NewFakeMounter([]mount.MountPoint{})
	server := newServerForTesting(t, mounter)

	volume, err := server.loadVolumeInfo(volumeId)
	assert.NoError(t, err)
	assert.NotNil(t, volume)
	assert.Equal(t, &metadata.Volume{}, volume)
}

func newServerForTesting(t *testing.T, mounter *mount.FakeMounter) CSIDriverServer {
	objects := []client.Object{
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   namespace,
				Labels: map[string]string{webhook.LabelInstance: dkName},
			},
		},
		&v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkName,
			},
		},
	}

	csiOptions := dtcsi.CSIOptions{RootDir: "/"}

	tmpFs := afero.NewMemMapFs()

	return CSIDriverServer{
		client:  fake.NewClient(objects...),
		log:     logr.TestLogger{T: t},
		opts:    csiOptions,
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
		db:      metadata.FakeMemoryDB(),
		fph:     metadata.FilePathHandler{RootDir: csiOptions.RootDir},
	}
}

func mockPublishedVolume(t *testing.T, server *CSIDriverServer) {
	mockOneAgent(t, server)
	err := server.db.InsertVolume(metadata.NewVolume(volumeId, podUID, agentVersion, tenantUuid))
	require.NoError(t, err)
	agentsVersionsMetric.WithLabelValues(agentVersion).Inc()
}

func mockOneAgent(t *testing.T, server *CSIDriverServer) {
	err := server.db.InsertTenant(metadata.NewTenant(tenantUuid, agentVersion, dkName))
	require.NoError(t, err)
}

func assertReferencesForPublishedVolume(t *testing.T, server *CSIDriverServer, mounter *mount.FakeMounter) {
	assert.NotEmpty(t, mounter.MountPoints)
	volume, err := server.loadVolumeInfo(volumeId)
	assert.NoError(t, err)
	assert.Equal(t, volume.VolumeID, volumeId)
	assert.Equal(t, volume.PodName, podUID)
	assert.Equal(t, volume.Version, agentVersion)
	assert.Equal(t, volume.TenantUUID, tenantUuid)
}

func assertNoReferencesForUnpublishedVolume(t *testing.T, server *CSIDriverServer) {
	volume, err := server.loadVolumeInfo(volumeId)
	assert.NoError(t, err)
	assert.Equal(t, &metadata.Volume{}, volume)
}

func resetMetrics() {
	agentsVersionsMetric.WithLabelValues(agentVersion).Set(0)
}
