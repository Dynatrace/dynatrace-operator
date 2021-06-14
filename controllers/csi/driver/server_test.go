package csidriver

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/container-storage-interface/spec/lib/go/csi"
	logr "github.com/go-logr/logr/testing"
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
		VolumeId: podUid,
		VolumeContext: map[string]string{
			podNamespaceContextKey: namespace,
			podUIDContextKey:       podUid,
		},
		TargetPath: "/path/to/container/filesystem/opt/dynatrace/oneagent",
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
		},
	}

	response, err := server.NodePublishVolume(context.TODO(), nodePublishVolumeRequest)

	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotEmpty(t, mounter.MountPoints)
}

func newServerForTesting(t *testing.T, mounter *mount.FakeMounter) CSIDriverServer {
	var err error

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

	osFs := afero.NewOsFs()
	tempDir, _ := afero.TempDir(osFs, "", "")
	tmpFs := afero.NewBasePathFs(osFs, tempDir)

	_ = tmpFs.MkdirAll(filepath.Join(tenantUuid), os.ModePerm)
	err = afero.WriteFile(tmpFs, filepath.Join(tenantUuid, dtcsi.VersionDir), []byte(agentVersion), fs.FileMode(0755))
	require.NoError(t, err)
	_ = tmpFs.MkdirAll(csiOptions.RootDir, os.ModePerm)
	err = afero.WriteFile(tmpFs, filepath.Join(csiOptions.RootDir, "tenant-"+dkName), []byte(tenantUuid), os.ModePerm)
	require.NoError(t, err)
	_ = tmpFs.MkdirAll(filepath.Join(csiOptions.RootDir, tenantUuid), os.ModePerm)
	err = afero.WriteFile(tmpFs, filepath.Join(csiOptions.RootDir, tenantUuid, "version"), []byte(agentVersion), os.ModePerm)
	require.NoError(t, err)
	_ = tmpFs.MkdirAll(filepath.Join(tenantUuid, dtcsi.GarbageCollectionPath, agentVersion), os.ModePerm)
	_ = tmpFs.MkdirAll(filepath.Join(dtcsi.GarbageCollectionPath), os.ModePerm)

	return CSIDriverServer{
		client:  fake.NewClient(objects...),
		log:     logr.NullLogger{},
		opts:    csiOptions,
		fs:      afero.Afero{Fs: tmpFs},
		mounter: mounter,
	}
}
