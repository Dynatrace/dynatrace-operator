package csidriver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bindConfig struct {
	agentDir                 string
	envDir                   string
	version                  string
	podToVersionReferenceDir string
}

func newBindConfig(ctx context.Context, svr *CSIDriverServer, volumeCfg *volumeConfig, fs afero.Afero) (*bindConfig, error) {
	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: volumeCfg.namespace}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Failed to query namespace %s: %s", volumeCfg.namespace, err.Error()))
	}

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Namespace '%s' doesn't have DynaKube assigned", volumeCfg.namespace))
	}

	tenantUUID, err := fs.ReadFile(filepath.Join(svr.opts.RootDir, fmt.Sprintf("tenant-%s", dkName)))
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("Failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	envDir := filepath.Join(svr.opts.RootDir, string(tenantUUID))

	versionBytes, err := fs.ReadFile(filepath.Join(envDir, "version"))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to query agent directory for DynaKube %s: %s", dkName, err.Error()))
	}

	version := string(versionBytes)
	agentDir := filepath.Join(envDir, "bin", version)

	podToVersionReferenceDir := filepath.Join(envDir, dtcsi.GarbageCollectionPath, string(versionBytes))
	if err := svr.fs.MkdirAll(podToVersionReferenceDir, os.ModePerm); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to create pod to version reference directory: %s", err))
	}

	return &bindConfig{
		agentDir:                 agentDir,
		envDir:                   envDir,
		version:                  version,
		podToVersionReferenceDir: podToVersionReferenceDir,
	}, nil
}
