package livenessprobe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/csitest/driver"
	mocks "github.com/Dynatrace/dynatrace-operator/test/mocks/github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNormalResponse(t *testing.T) {
	csiAddress, cleanUpFunc := launchCSIServer(t, 0)
	defer cleanUpFunc()

	livenessprobeServer := &Server{
		csiAddress:   csiAddress,
		probeTimeout: time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/healthz" {
			livenessprobeServer.probeRequest(rw, req)
		}
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/healthz", server.URL), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDelayedResponse(t *testing.T) {
	csiAddress, cleanUpFunc := launchCSIServer(t, 5*time.Second)
	defer cleanUpFunc()

	livenessprobeServer := &Server{
		csiAddress:   csiAddress,
		probeTimeout: time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/healthz" {
			livenessprobeServer.probeRequest(rw, req)
		}
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/healthz", server.URL), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func launchCSIServer(t *testing.T, probeTimeout time.Duration) (string, func()) {
	driver, idServer, cleanUpFunc := createMockServer(t)

	var injectedErr error

	outProbe := &csi.ProbeResponse{}
	idServer.EXPECT().Probe(mock.Anything, mock.Anything).WaitUntil(time.After(probeTimeout)).Return(outProbe, injectedErr).Times(1)

	return driver.Address(), cleanUpFunc
}

func createMockServer(t *testing.T) (
	*driver.MockCSIDriver,
	*mocks.IdentityServer,
	func()) {
	identityServer := mocks.NewIdentityServer(t)
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
		drv.Stop()
		os.RemoveAll(csiEndpoint)
	}
}
