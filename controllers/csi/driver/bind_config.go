package csidriver

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/codemodules"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bindConfig struct {
	agentDir string
	envDir   string
	version  string
}

func newBindConfig(ctx context.Context, svr *CSIDriverServer, volumeCfg *volumeConfig,
	readFileFunc func(filename string) ([]byte, error),
	mkDirFunc func(path string, perm fs.FileMode) error) (*bindConfig, error) {
	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: volumeCfg.namespace}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Failed to query namespace %s: %s", volumeCfg.namespace, err.Error()))
	}

	dk, err := codemodules.FindForNamespace(ctx, svr.client, &ns)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if dk == nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Namespace '%s' doesn't have DynaKube assigned", volumeCfg.namespace))
	}

	tenantUUID, err := readFileFunc(filepath.Join(svr.opts.RootDir, dtcsi.DataPath, fmt.Sprintf("tenant-%s", dk.Name)))
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("Failed to extract tenant for DynaKube %s: %s", dk.Name, err.Error()))
	}
	envDir := filepath.Join(svr.opts.RootDir, dtcsi.DataPath, string(tenantUUID))

	for _, dir := range []string{
		filepath.Join(envDir, "log", volumeCfg.podUID),
		filepath.Join(envDir, "datastorage", volumeCfg.podUID),
	} {
		if err = mkDirFunc(dir, 0770); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	ver, err := readFileFunc(filepath.Join(envDir, "version"))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to query agent directory for DynaKube %s: %s", dk.Name, err.Error()))
	}

	agentDir := filepath.Join(envDir, "bin", string(ver))

	return &bindConfig{
		agentDir: agentDir,
		envDir:   envDir,
		version:  string(ver),
	}, nil
}
