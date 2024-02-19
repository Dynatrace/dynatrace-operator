package dynakube

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	errorCSIRequired = `The Dynakube's specification requires the CSI driver to work. Make sure you deployed the correct manifests.
`
	errorCSIEnabledRequired = `The Dynakube's specification specifies readonly-CSI volume, but the CSI driver is not enabled.
`
)

func missingCSIDaemonSet(ctx context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedsCSIDriver() {
		return ""
	}

	csiDaemonSet := appsv1.DaemonSet{}

	err := dv.clt.Get(ctx, types.NamespacedName{Name: dtcsi.DaemonSetName, Namespace: dynakube.Namespace}, &csiDaemonSet)
	if k8serrors.IsNotFound(err) {
		log.Info("requested dynakube uses csi driver, but csi driver is missing in the cluster", "name", dynakube.Name, "namespace", dynakube.Namespace)

		return errorCSIRequired
	} else if err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())
	}

	return ""
}

func disabledCSIForReadonlyCSIVolume(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedsCSIDriver() && dynakube.FeatureReadOnlyCsiVolume() {
		log.Info("requested dynakube uses readonly csi volume, but csi driver is not enabled", "name", dynakube.Name, "namespace", dynakube.Namespace)

		return errorCSIEnabledRequired
	}

	return ""
}
