package registrar

import (
	"fmt"
	"net"
	"os"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

var (
	log = logd.Get().WithName("csi-registrar")
)

type Server struct {
	driverName             string
	csiAddress             string
	endpoint               string
	pluginRegistrationPath string
	version                []string
}

var _ registerapi.RegistrationServer = Server{}

func NewServer(driverName string, csiAddress string, endpoint string, pluginRegistrationPath string, versions []string) *Server {
	return &Server{
		driverName:             driverName,
		csiAddress:             csiAddress,
		endpoint:               endpoint,
		pluginRegistrationPath: pluginRegistrationPath,
		version:                versions,
	}
}

func (srv Server) Start(ctx context.Context) error {
	log.Info("starting registrar")

	if err := srv.isDriverRunning(ctx); err != nil {
		return err
	}

	socketPath := srv.buildRegistrationDir()
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

func (srv Server) buildRegistrationDir() string {
	// registrar.VolumeMounts.registration-dir
	return fmt.Sprintf("%s/%s-reg.sock", srv.pluginRegistrationPath, dtcsi.DriverName)
}

func removeExistingSocketFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove existing %s socket file", path)
	}

	return nil
}

func (srv Server) isDriverRunning(ctx context.Context) error {
	conn, err := connection.Connect(ctx, srv.csiAddress, nil, connection.WithTimeout(0))
	if err != nil {
		log.Error(err, "failed to establish connection to CSI driver")

		return err
	}

	driverName, err := rpc.GetDriverName(ctx, conn)
	conn.Close()

	if err != nil {
		log.Error(err, "failed to get driver name")

		return err
	}

	log.Info("CSI driver is running", "driver name", driverName)

	return nil
}
