package registrar

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-test/v5/driver"
	"github.com/kubernetes-csi/csi-test/v5/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

const (
	driverName                  = "test-driver-name"
	testKubeletRegistrationPath = "/test-kubelet-registration-path/csi.sock"
)

var (
	testVersions = []string{"2.0.0", "3.0.0"}
)

func TestPluginInfoResponse(t *testing.T) {
	csiAddress, cleanUpFunc := testLaunchCSIServer(t)
	defer cleanUpFunc()

	var wg sync.WaitGroup

	wg.Add(1)

	pluginRegistrationPath := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	server := launchTestRegistrarServer(t, ctx, &wg, csiAddress, pluginRegistrationPath)

	time.Sleep(time.Second)

	conn, err := connection.Connect(ctx, server.buildRegistrationDir(), nil, connection.WithTimeout(0))
	require.NoError(t, err)

	client := registerapi.NewRegistrationClient(conn)
	req := registerapi.InfoRequest{}
	pluginInfo, err := client.GetInfo(ctx, &req)
	require.NoError(t, err)

	assert.Equal(t, driverName, pluginInfo.GetName())
	assert.Equal(t, registerapi.CSIPlugin, pluginInfo.GetType())
	assert.Equal(t, testKubeletRegistrationPath, pluginInfo.GetEndpoint())
	assert.Equal(t, testVersions, pluginInfo.GetSupportedVersions())

	cancel()
	wg.Wait()
}

func launchTestRegistrarServer(t *testing.T, ctx context.Context, wg *sync.WaitGroup, endpoint string, pluginRegistrationPath string) *Server {
	server := NewServer(driverName, endpoint, testKubeletRegistrationPath, pluginRegistrationPath, testVersions)

	go func() {
		err := server.Start(ctx)
		t.Log("stopped registrar server", "err", err)

		wg.Done()
	}()

	return server
}

func testLaunchCSIServer(t *testing.T) (string, func()) {
	driver, idServer, cleanUpFunc := createMockServer(t)

	var injectedErr error

	inPluginInfo := &csi.GetPluginInfoRequest{}
	outPluginInfo := &csi.GetPluginInfoResponse{
		Name:          driverName,
		VendorVersion: "test",
	}
	idServer.EXPECT().GetPluginInfo(gomock.Any(), utils.Protobuf(inPluginInfo)).Return(outPluginInfo, injectedErr).Times(1)

	return driver.Address(), cleanUpFunc
}

func createMockServer(t *testing.T) (
	*driver.MockCSIDriver,
	*driver.MockIdentityServer,
	func()) {
	// Start the mock server
	mockController := gomock.NewController(t)
	identityServer := driver.NewMockIdentityServer(mockController)
	drv := driver.NewMockCSIDriver(&driver.MockCSIDriverServers{
		Identity: identityServer,
	})

	tmpDir := t.TempDir()

	csiEndpoint := fmt.Sprintf("%s/csi.sock", tmpDir)
	err := drv.StartOnAddress("unix", csiEndpoint)

	if err != nil {
		t.Errorf("failed to start the csi driver at %s: %v", csiEndpoint, err)
	}

	return drv, identityServer, func() {
		mockController.Finish()
		drv.Stop()
		os.RemoveAll(csiEndpoint)
	}
}
