package livenessprobe

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	driverName = "test-driver-name"
	healthPort = "9808"
)

type testCSIServer struct {
	csi.UnimplementedIdentityServer

	probeTimeout time.Duration

	pluginInfoReqCount int
	probeReqCount      int
}

func TestNormalResponse(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(2)

	endpoint := t.TempDir() + "/csi.sock"

	ctx, cancel := context.WithCancel(context.Background())

	csiServer, err := launchTestCSIServer(t, ctx, &wg, "unix://"+endpoint, 0)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	server := launchTestLivenessprobeServer(t, ctx, &wg, endpoint, 1*time.Second)
	time.Sleep(1 * time.Second)

	resp, err := http.Get("http://127.0.0.1:" + server.healthPort + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, 1, csiServer.pluginInfoReqCount)
	assert.Equal(t, 1, csiServer.probeReqCount)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://127.0.0.1:" + server.healthPort + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, 1, csiServer.pluginInfoReqCount)
	assert.Equal(t, 2, csiServer.probeReqCount)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	wg.Wait()
}

func TestDelayedResponse(t *testing.T) {
	// it's not possible to register the same endpoint twice to the same mux
	http.DefaultServeMux = http.NewServeMux()

	var wg sync.WaitGroup

	wg.Add(2)

	endpoint := t.TempDir() + "/csi.sock"

	ctx, cancel := context.WithCancel(context.Background())

	csiServer, err := launchTestCSIServer(t, ctx, &wg, "unix://"+endpoint, 2*time.Second)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	server := launchTestLivenessprobeServer(t, ctx, &wg, endpoint, 1*time.Second)
	time.Sleep(1 * time.Second)

	resp, err := http.Get("http://127.0.0.1:" + server.healthPort + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, 1, csiServer.pluginInfoReqCount)
	assert.Equal(t, 1, csiServer.probeReqCount)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	cancel()
	wg.Wait()
}

func launchTestLivenessprobeServer(t *testing.T, ctx context.Context, wg *sync.WaitGroup, csiAddress string, probeTimeout time.Duration) *Server {
	server := NewServer(driverName, csiAddress, healthPort, probeTimeout.String())

	go func() {
		err := server.Start(ctx)
		t.Log("stopped HTTP server", "err", err)

		wg.Done()
	}()

	return server
}

func launchTestCSIServer(t *testing.T, ctx context.Context, wg *sync.WaitGroup, csiAddress string, probeTimeout time.Duration) (*testCSIServer, error) {
	csiServer := &testCSIServer{
		probeTimeout: probeTimeout,
	}

	endpoint, err := url.Parse(csiAddress)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to parse endpoint '%s'", csiAddress))
	}

	addr := endpoint.Host + endpoint.Path

	listener, err := net.Listen(endpoint.Scheme, addr)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to start testCSIServer")
	}

	server := grpc.NewServer()

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	csi.RegisterIdentityServer(server, csiServer)

	go func() {
		server.Serve(listener)
		t.Log("stopped CSI server", "err", err)

		wg.Done()
	}()

	return csiServer, nil
}

func (srv *testCSIServer) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	srv.pluginInfoReqCount++

	return &csi.GetPluginInfoResponse{Name: driverName, VendorVersion: "test"}, nil
}

func (srv *testCSIServer) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	srv.probeReqCount++

	if srv.probeTimeout > 0 {
		time.Sleep(srv.probeTimeout)
	}

	return &csi.ProbeResponse{}, nil
}
