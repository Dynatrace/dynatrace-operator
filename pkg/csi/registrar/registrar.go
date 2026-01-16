package registrar

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

const (
	permAllUG = 0077
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

func (s Server) Start(ctx context.Context) error {
	log.Info("starting registrar")

	if err := s.isDriverRunning(ctx); err != nil {
		return err
	}

	socketPath := s.buildRegistrationDir()
	log.Info("socket path", "path", s.pluginRegistrationPath)

	if err := removeExistingSocketFile(socketPath); err != nil {
		log.Error(err, "failed to clean up socket file")

		return err
	}

	defer removeExistingSocketFile(socketPath)

	var oldmask int
	if runtime.GOOS == "linux" {
		// Default to only user accessible socket, caller can open up later if desired
		oldmask = unix.Umask(permAllUG)
	}

	lis, err := (&net.ListenConfig{}).Listen(ctx, "unix", socketPath)
	if err != nil {
		log.Error(err, "failed to listen on socket "+socketPath)

		return err
	}

	if runtime.GOOS == "linux" {
		unix.Umask(oldmask)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(grpcMessageLogger()))

	registerapi.RegisterRegistrationServer(server, s)

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

func (s Server) GetInfo(_ context.Context, req *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	log.Info("received GetInfo", "request", req)

	return &registerapi.PluginInfo{
		Type:              registerapi.CSIPlugin,
		Name:              s.driverName,
		Endpoint:          s.endpoint,
		SupportedVersions: s.version,
	}, nil
}

func (s Server) NotifyRegistrationStatus(_ context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	log.Info("received NotifyRegistrationStatus", "status", status)

	if !status.PluginRegistered {
		log.Error(errors.New("plugin registration failed"), "plugin registration failed", "error", status.Error)
	}

	return &registerapi.RegistrationStatusResponse{}, nil
}

func grpcMessageLogger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log.Debug("GRPC call", "full_method", info.FullMethod)

		resp, err := handler(ctx, req)
		if err != nil {
			log.Info("GRPC failed", "full_method", info.FullMethod, "err", err.Error())
		}

		return resp, err
	}
}

func (s Server) buildRegistrationDir() string {
	// registrar.VolumeMounts.registration-dir
	return fmt.Sprintf("%s/%s-reg.sock", s.pluginRegistrationPath, dtcsi.DriverName)
}

func removeExistingSocketFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove existing %s socket file", path)
	}

	return nil
}

func (s Server) isDriverRunning(ctx context.Context) error {
	conn, err := connection.Connect(ctx, s.csiAddress, nil, connection.WithTimeout(0))
	if err != nil {
		log.Error(err, "failed to establish connection to CSI driver")

		return err
	}
	defer conn.Close()

	driverName, err := rpc.GetDriverName(ctx, conn)
	if err != nil {
		log.Error(err, "failed to get driver name")

		return err
	}

	log.Info("CSI driver is running", "driver name", driverName)

	return nil
}
