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

func (srv Server) Start(ctx context.Context) error {
	log.Info("starting registrar")

	socketPath := buildRegistrationDir()
	if err := removeExistingSocketFile(socketPath); err != nil {
		log.Error(err, "failed to clean up socket file")

		return err
	}
	defer removeExistingSocketFile(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Error(err, "failed to listen on socket")

		return err
	}

	server := grpc.NewServer()

	registerapi.RegisterRegistrationServer(server, srv)

	log.Info("starting registration server", "address", lis.Addr())

	go func() {
		<-ctx.Done()
		log.Info("stopping server")
		server.GracefulStop()
		log.Info("stopped server")
	}()

	if err := server.Serve(lis); err != nil {
		log.Error(err, "failed to serve")

		return err
	}

	return nil
}

func (srv Server) GetInfo(_ context.Context, req *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	log.Info("received GetInfo", "request", req)

	return &registerapi.PluginInfo{
		Type:              registerapi.CSIPlugin,
		Name:              srv.driverName,
		Endpoint:          srv.endpoint,
		SupportedVersions: srv.version,
	}, nil
}

func (srv Server) NotifyRegistrationStatus(_ context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	log.Info("received NotifyRegistrationStatus", "status", status)

	if !status.PluginRegistered {
		log.Error(errors.New("plugin registration failed"), "plugin registration failed", "error", status.Error)
	}

	return &registerapi.RegistrationStatusResponse{}, nil
}

func buildRegistrationDir() string {
	// registrar.VolumeMounts.registration-dir
	return fmt.Sprintf("/registration/%s-reg.sock", dtcsi.DriverName)
}

func removeExistingSocketFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove existing %s socket file", path)
	}

	return nil
}
