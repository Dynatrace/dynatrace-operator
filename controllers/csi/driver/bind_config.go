package csidriver

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/webhook"
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

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Namespace '%s' doesn't have DynaKube assigned", volumeCfg.namespace))
	}

	envID, err := readFileFunc(filepath.Join(svr.opts.RootDir, dtcsi.DataPath, fmt.Sprintf("tenant-%s", dkName)))
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("Failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	envDir := filepath.Join(svr.opts.RootDir, dtcsi.DataPath, string(envID))

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
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to query agent directory for DynaKube %s: %s", dkName, err.Error()))
	}

	agentDir := filepath.Join(envDir, "bin", string(ver)+"-"+volumeCfg.flavor)

	return &bindConfig{
		agentDir: agentDir,
		envDir:   envDir,
		version:  string(ver),
	}, nil
}
