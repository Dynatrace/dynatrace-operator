package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func prepareVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{getOneAgentSecretVolumeMount(), getNodeMetadataVolumeMount()}

	if dk.OneAgent().IsReadOnlyFSSupported() {
		volumeMounts = append(volumeMounts, getReadOnlyRootMount())
		if dk.OneAgent().IsCSIAvailable() {
			volumeMounts = append(volumeMounts, getCSIStorageMount())
		} else {
			volumeMounts = append(volumeMounts, getStorageVolumeMount(dk))
		}
	} else {
		volumeMounts = append(volumeMounts, getRootMount())
	}

	if dk.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getClusterCaCertVolumeMount())
	}

	if dk.ActiveGate().HasCaCert() {
		volumeMounts = append(volumeMounts, getActiveGateCaCertVolumeMount())
	}

	if dk.NeedsOneAgentProxy() {
		volumeMounts = append(volumeMounts, getHTTPProxyMount())
	}

	return volumeMounts
}

func getNodeMetadataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      nodeMetadataVolumeName,
		ReadOnly:  true,
		MountPath: nodeMetadataFilePath,
		SubPath:   nodeMetadataFilename,
	}
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

func getStorageVolumeMount(dk *dynakube.DynaKube) corev1.VolumeMount {
	// the TenantUUID is already set
	tenant, _ := dk.TenantUUID()

	return corev1.VolumeMount{
		Name:      storageVolumeName,
		SubPath:   tenant,
		MountPath: csiStorageVolumeMount,
	}
}

func getHTTPProxyMount() corev1.VolumeMount {
	return proxy.BuildVolumeMount()
}

func prepareVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume(), getNodeMetadataVolume(), getOneAgentSecretVolume(dk)}

	if dk.OneAgent().IsReadOnlyFSSupported() {
		if dk.OneAgent().IsCSIAvailable() {
			volumes = append(volumes, getCSIStorageVolume(dk))
		} else {
			volumes = append(volumes, getStorageVolume(dk))
		}
	}

	if dk.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(dk))
	}

	if dk.ActiveGate().HasCaCert() {
		volumes = append(volumes, getActiveGateCaCertVolume(dk))
	}

	if dk.NeedsOneAgentProxy() {
		volumes = append(volumes, buildHTTPProxyVolume(dk))
	}

	return volumes
}

func getNodeMetadataVolume() corev1.Volume {
	return corev1.Volume{
		Name: nodeMetadataVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
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
				},
			},
		},
	}
}

func getStorageVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: storageVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: dk.OneAgent().GetHostPath(),
				Type: ptr.To(corev1.HostPathDirectoryOrCreate),
			},
		},
	}
}

func getActiveGateCaCertVolume(dk *dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: activeGateCaCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.Spec.ActiveGate.GetTLSSecretName(),
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

func buildHTTPProxyVolume(dk *dynakube.DynaKube) corev1.Volume {
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
				SecretName: dk.OneAgent().GetTenantSecret(),
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
