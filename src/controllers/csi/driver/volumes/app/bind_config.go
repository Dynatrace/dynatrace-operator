package appvolumes

import (
	"context"
	"fmt"

	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bindConfig struct {
	tenantUUID string
	version    string
}

func newBindConfig(ctx context.Context, svr *AppVolumePublisher, volumeCfg *csivolumes.VolumeConfig) (*bindConfig, error) {
	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: volumeCfg.Namespace}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to query namespace %s: %s", volumeCfg.Namespace, err.Error()))
	}

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("namespace '%s' doesn't have DynaKube assigned", volumeCfg.Namespace))
	}

	dynakube, err := svr.db.GetDynakube(dkName)
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	if dynakube == nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("dynakube (%s) is missing from metadata database", dkName))
	}
	return &bindConfig{
		tenantUUID: dynakube.TenantUUID,
		version:    dynakube.LatestVersion,
	}, nil
}
