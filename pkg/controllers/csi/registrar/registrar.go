package registrar

import (
	"fmt"
	"net"
	"os"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

var (
	log = logd.Get().WithName("csi-registrar")
)

type Server struct {
	driverName string
	endpoint   string
	version    []string
}

var _ registerapi.RegistrationServer = Server{}

func NewServer(driverName string, endpoint string, versions []string) *Server {
	return &Server{
		driverName: driverName,
		endpoint:   endpoint,
		version:    versions,
	}
}

func (svr *Server) Start(ctx context.Context) error {
	log.Info("starting registrar")

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	socketPath := buildRegistrationSocketPath()
	if err := CleanupSocketFile(socketPath); err != nil {
		log.Error(err, "failed to clean up socket file")

		return err
	}

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Error(err, "failed to listen on socket")

		return err
	}

	server := grpc.NewServer()

	registerapi.RegisterRegistrationServer(server, svr)

	log.Info("starting registration server", "address", lis.Addr())

	if err := server.Serve(lis); err != nil {
		log.Error(err, "failed to serve")
	}

	<-cancelCtx.Done()
	server.GracefulStop()

	return nil
}

func buildRegistrationSocketPath() string {
	return fmt.Sprintf("/registration/%s-reg.sock", dtcsi.DriverName)
}

// GetInfo is the RPC invoked by plugin watcher
func (e Server) GetInfo(ctx context.Context, req *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	log.Info("Received GetInfo call", "request", req)

	return &registerapi.PluginInfo{
		Type:              registerapi.CSIPlugin,
		Name:              e.driverName,
		Endpoint:          e.endpoint,
		SupportedVersions: e.version,
	}, nil
}

func (e Server) NotifyRegistrationStatus(ctx context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	log.Info("Received NotifyRegistrationStatus call", "status", status)

	if !status.PluginRegistered {
		log.Error(errors.New("plugin registration failed"), "plugin registration failed", "error", status.Error)
	}

	return &registerapi.RegistrationStatusResponse{}, nil
}

func CleanupSocketFile(socketPath string) error {
	socketExists, err := DoesSocketExist(socketPath)
	if err != nil {
		return err
	}

	if socketExists {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("failed to remove stale socket %s with error: %w", socketPath, err)
		}
	}

	return nil
}

func DoesSocketExist(socketPath string) (bool, error) {
	fi, err := os.Stat(socketPath)
	if err == nil {
		if isSocket := fi.Mode()&os.ModeSocket != 0; isSocket {
			return true, nil
		}

		return false, fmt.Errorf("file exists in socketPath %s but it's not a socket.: %+v", socketPath, fi)
	}

	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to stat the socket %s with error: %w", socketPath, err)
	}

	return false, nil
}
