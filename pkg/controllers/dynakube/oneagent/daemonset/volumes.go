package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	corev1 "k8s.io/api/core/v1"
)

func prepareVolumeMounts(instance *dynatracev1beta1.DynaKube) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	volumeMounts = append(volumeMounts, getOneAgentSecretVolumeMount())

	if instance != nil && instance.NeedsReadOnlyOneAgents() {
		volumeMounts = append(volumeMounts, getReadOnlyRootMount())
		volumeMounts = append(volumeMounts, getCSIStorageMount())
	} else {
		volumeMounts = append(volumeMounts, getRootMount())
	}

	if instance != nil && instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getClusterCaCertVolumeMount())
	}

	if instance != nil && instance.HasActiveGateCaCert() {
		volumeMounts = append(volumeMounts, getActiveGateCaCertVolumeMount())
	}

	if instance != nil && instance.HasProxy() {
		volumeMounts = append(volumeMounts, getHttpProxyMount())
	}

	return volumeMounts
}

func getClusterCaCertVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      clusterCaCertVolumeName,
		MountPath: clusterCaCertVolumeMountPath,
	}
}

func getActiveGateCaCertVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      activeGateCaCertVolumeName,
		MountPath: activeGateCaCertVolumeMountPath,
	}
}

func getOneAgentSecretVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      connectioninfo.TenantSecretVolumeName,
		ReadOnly:  true,
		MountPath: connectioninfo.TenantTokenMountPoint,
		SubPath:   connectioninfo.TenantTokenKey,
	}
}

func getRootMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      hostRootVolumeName,
		MountPath: hostRootVolumeMountPath,
	}
}

func getReadOnlyRootMount() corev1.VolumeMount {
	rootMount := getRootMount()
	rootMount.ReadOnly = true

	return rootMount
}

func getCSIStorageMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      csiStorageVolumeName,
		MountPath: csiStorageVolumeMount,
	}
}

func getHttpProxyMount() corev1.VolumeMount {
	return proxy.BuildVolumeMount()
}

func prepareVolumes(instance *dynatracev1beta1.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if instance == nil {
		return volumes
	}

	volumes = append(volumes, getOneAgentSecretVolume(instance))

	if instance.NeedsReadOnlyOneAgents() {
		volumes = append(volumes, getCSIStorageVolume(instance))
	}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(instance))
	}

	if instance.HasActiveGateCaCert() {
		volumes = append(volumes, getActiveGateCaCertVolume(instance))
	}

	if instance.HasProxy() {
		volumes = append(volumes, buildHttpProxyVolume(instance))
	}

	return volumes
}

func getCertificateVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: clusterCaCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: instance.Spec.TrustedCAs,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "certs",
						Path: "certs.pem",
					},
				},
			},
		},
	}
}

func getCSIStorageVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: csiStorageVolumeName,
		VolumeSource: corev1.VolumeSource{
			CSI: &corev1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
				VolumeAttributes: map[string]string{
					csivolumes.CSIVolumeAttributeModeField:     hostvolumes.Mode,
					csivolumes.CSIVolumeAttributeDynakubeField: instance.Name,
				},
			},
		},
	}
}

func getActiveGateCaCertVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: activeGateCaCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Spec.ActiveGate.TlsSecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  "server.crt",
						Path: "custom.pem",
					},
				},
			},
		},
	}
}

func buildHttpProxyVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: proxy.SecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: proxy.BuildSecretName(instance.Name),
			},
		},
	}
}

func getOneAgentSecretVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: connectioninfo.TenantSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.OneagentTenantSecret(),
			},
		},
	}
}

func getRootVolume() corev1.Volume {
	return corev1.Volume{
		Name: hostRootVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/",
			},
		},
	}
}
