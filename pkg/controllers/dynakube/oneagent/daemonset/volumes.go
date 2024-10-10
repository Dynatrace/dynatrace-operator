package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	corev1 "k8s.io/api/core/v1"
)

func prepareVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	volumeMounts = append(volumeMounts, getOneAgentSecretVolumeMount())

	if dk != nil && dk.NeedsReadOnlyOneAgents() {
		volumeMounts = append(volumeMounts, getReadOnlyRootMount())
		volumeMounts = append(volumeMounts, getCSIStorageMount())
	} else {
		volumeMounts = append(volumeMounts, getRootMount())
	}

	if dk != nil && dk.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getClusterCaCertVolumeMount())
	}

	if dk != nil && dk.ActiveGate().HasCaCert() {
		volumeMounts = append(volumeMounts, getActiveGateCaCertVolumeMount())
	}

	if dk != nil && dk.NeedsOneAgentProxy() {
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

func prepareVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if dk == nil {
		return volumes
	}

	volumes = append(volumes, getOneAgentSecretVolume(dk))

	if dk.NeedsReadOnlyOneAgents() {
		volumes = append(volumes, getCSIStorageVolume(dk))
	}

	if dk.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(dk))
	}

	if dk.ActiveGate().HasCaCert() {
		volumes = append(volumes, getActiveGateCaCertVolume(dk))
	}

	if dk.NeedsOneAgentProxy() {
		volumes = append(volumes, buildHttpProxyVolume(dk))
	}

	return volumes
}

func getCertificateVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: clusterCaCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: dk.Spec.TrustedCAs,
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

func getCSIStorageVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: csiStorageVolumeName,
		VolumeSource: corev1.VolumeSource{
			CSI: &corev1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
				VolumeAttributes: map[string]string{
					csivolumes.CSIVolumeAttributeModeField:     hostvolumes.Mode,
					csivolumes.CSIVolumeAttributeDynakubeField: dk.Name,
					csivolumes.CSIVolumeAttributeRetryTimeout:  dk.FeatureMaxCSIRetryTimeout().String(),
				},
			},
		},
	}
}

func getActiveGateCaCertVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: activeGateCaCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.Spec.ActiveGate.TlsSecretName,
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

func buildHttpProxyVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: proxy.SecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: proxy.BuildSecretName(dk.Name),
			},
		},
	}
}

func getOneAgentSecretVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: connectioninfo.TenantSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.OneagentTenantSecret(),
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
