package livenessprobe

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-csi/csi-test/v5/driver"
	"github.com/kubernetes-csi/csi-test/v5/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

const (
	driverName = "test-driver-name"
	healthPort = "9808"
)

func TestNormalResponse(t *testing.T) {
	csiAddress, cleanUpFunc := testLaunchCSIServer(t, 0)
	defer cleanUpFunc()

	var wg sync.WaitGroup

	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())

	server := testLaunchLivenessprobeServer(t, ctx, &wg, csiAddress, time.Second)
	time.Sleep(time.Second)

	resp, err := http.Get("http://127.0.0.1:" + server.healthPort + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	wg.Wait()
}

func TestDelayedResponse(t *testing.T) {
	csiAddress, cleanUpFunc := testLaunchCSIServer(t, 2*time.Second)
	defer cleanUpFunc()

	// it's not possible to register the same endpoint twice to the same mux
	http.DefaultServeMux = http.NewServeMux()

	var wg sync.WaitGroup

	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())

	server := testLaunchLivenessprobeServer(t, ctx, &wg, csiAddress, time.Second)
	time.Sleep(time.Second)

	resp, err := http.Get("http://127.0.0.1:" + server.healthPort + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	cancel()
	wg.Wait()
}

func testLaunchLivenessprobeServer(t *testing.T, ctx context.Context, wg *sync.WaitGroup, csiAddress string, probeTimeout time.Duration) *Server {
	server := NewServer(driverName, csiAddress, healthPort, probeTimeout)

	go func() {
		err := server.Start(ctx)
		t.Log("stopped HTTP server", "err", err)

		wg.Done()
	}()

	return server
}

func testLaunchCSIServer(t *testing.T, probeTimeout time.Duration) (string, func()) {
	driver, idServer, cleanUpFunc := createMockServer(t)

	var injectedErr error

	inProbe := &csi.ProbeRequest{}
	outProbe := &csi.ProbeResponse{}
	idServer.EXPECT().Probe(gomock.Any(), utils.Protobuf(inProbe)).Return(outProbe, injectedErr).Times(1).Do(func(any, any) { time.Sleep(probeTimeout) })

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
