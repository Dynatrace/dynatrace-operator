package csidriver

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	_ "k8s.io/component-base/logs/json/register"
	"k8s.io/klog/v2"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

// registrationServer is a sample plugin to work with plugin watcher
type registrationServer struct {
	driverName string
	endpoint   string
	version    []string
}

var _ registerapi.RegistrationServer = registrationServer{}

// newRegistrationServer returns an initialized registrationServer instance
func newRegistrationServer(driverName string, endpoint string, versions []string) registerapi.RegistrationServer {
	return &registrationServer{
		driverName: driverName,
		endpoint:   endpoint,
		version:    versions,
	}
}

// GetInfo is the RPC invoked by plugin watcher
func (e registrationServer) GetInfo(ctx context.Context, req *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	logger := klog.FromContext(ctx)
	logger.Info("Received GetInfo call", "request", req)

	return &registerapi.PluginInfo{
		Type:              registerapi.CSIPlugin,
		Name:              e.driverName,
		Endpoint:          e.endpoint,
		SupportedVersions: e.version,
	}, nil
}

func (e registrationServer) NotifyRegistrationStatus(ctx context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
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
