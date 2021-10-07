package csidriver

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bindConfig struct {
	tenantUUID string
	version    string
}

func newBindConfig(ctx context.Context, svr *CSIDriverServer, volumeCfg *volumeConfig) (*bindConfig, error) {
	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: volumeCfg.namespace}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to query namespace %s: %s", volumeCfg.namespace, err.Error()))
	}

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("namespace '%s' doesn't have DynaKube assigned", volumeCfg.namespace))
	}

	tenant, err := svr.db.GetTenant(dkName)
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	if tenant == nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("tenant is missing from metadata for DynaKube %s", dkName))
	}
	return &bindConfig{
		tenantUUID: tenant.TenantUUID,
		version:    tenant.LatestVersion,
	}, nil
}
