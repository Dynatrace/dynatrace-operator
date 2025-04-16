package registrar

import (
	"fmt"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

const (
	driverName                  = "test-driver-name"
	testKubeletRegistrationPath = "/test-kubelet-registration-path/csi.sock"
)

var (
	testVersions = []string{"2.0.0", "3.0.0"}
)

type testCSIServer struct {
	csi.UnimplementedIdentityServer

	probeTimeout time.Duration
}

func TestPluginInfoResponse(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(2)

	csiAddress := t.TempDir() + "/csi.sock"
	pluginRegistrationPath := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	_, err := launchTestCSIServer(t, ctx, &wg, "unix://"+csiAddress, 0)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	server := launchTestRegistrarServer(t, ctx, &wg, csiAddress, pluginRegistrationPath)

	time.Sleep(1 * time.Second)

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
	return &csi.GetPluginInfoResponse{Name: driverName, VendorVersion: "test"}, nil
}
