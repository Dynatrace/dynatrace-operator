package persistentvolumeclaimspec

import corev1 "k8s.io/api/core/v1"

func IsEmpty(pvcs *corev1.PersistentVolumeClaimSpec) bool {
	if pvcs != nil && (len(pvcs.AccessModes) > 0 ||
		pvcs.Selector != nil ||
		len(pvcs.Resources.Limits) > 0 ||
		len(pvcs.Resources.Requests) > 0 ||
		len(pvcs.VolumeName) > 0 ||
		pvcs.StorageClassName != nil ||
		pvcs.VolumeMode != nil ||
		pvcs.DataSource != nil ||
		pvcs.DataSourceRef != nil ||
		pvcs.VolumeAttributesClassName != nil) {
		return false
	}

	return true
}
