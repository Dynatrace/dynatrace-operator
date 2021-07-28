package csidriver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/storage"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bindConfig struct {
	tenantUUID string
	agentDir   string
	envDir     string
	version    string
}

func newBindConfig(ctx context.Context, svr *CSIDriverServer, volumeCfg *volumeConfig, db storage.Access) (*bindConfig, error) {
	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: volumeCfg.namespace}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to query namespace %s: %s", volumeCfg.namespace, err.Error()))
	}

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("namespace '%s' doesn't have DynaKube assigned", volumeCfg.namespace))
	}

	tenant, err := db.GetTenantViaDynakube(dkName)
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	if tenant == nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("tenant is missing from storage for DynaKube %s", dkName))
	}
	envDir := filepath.Join(svr.opts.RootDir, tenant.UUID)
	agentDir := filepath.Join(envDir, "bin", tenant.LatestVersion)
	return &bindConfig{
		tenantUUID: tenant.UUID,
		agentDir:   agentDir,
		envDir:     envDir,
		version:    tenant.LatestVersion,
	}, nil
}
