package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorCSIEnabledRequired = `The Dynakube's specification specifies readonly-CSI volume, but the CSI driver is not enabled.
`
)

func disabledCSIForReadonlyCSIVolume(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	isCSINotUsed := !dk.IsCSIAvailable() || (!isCSIRequired(dk) && !isCSIOptional(dk))
	if dk.FeatureReadOnlyCsiVolume() && isCSINotUsed {
		log.Info("requested dynakube uses readonly csi volume, but csi driver is not enabled", "name", dk.Name, "namespace", dk.Namespace)

		return errorCSIEnabledRequired
	}

	return ""
}

// IsCSIDriverOptional checks if the DynaKube may use the csi-driver if available, otherwise fallbacks exist to provide similar functionality.
func isCSIOptional(dk *dynakube.DynaKube) bool {
	return dk.HostMonitoringMode() || dk.ApplicationMonitoringMode()
}
