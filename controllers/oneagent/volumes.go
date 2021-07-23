package oneagent

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func prepareVolumeMounts(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) []corev1.VolumeMount {
	rootMount := getRootMount()
	var volumeMounts []corev1.VolumeMount

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getCertificateMount())
	}

	if fs.ReadOnly.Enabled {
		volumeMounts = append(volumeMounts, getInstallationMount())
		rootMount.ReadOnly = true
	}

	volumeMounts = append(volumeMounts, rootMount)
	return volumeMounts
}

func getInstallationMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      oneagentInstallationMountName,
		MountPath: oneagentInstallationMountPath,
	}
}

func getCertificateMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "certs",
		MountPath: "/mnt/dynatrace/certs",
	}
}

func getRootMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      hostRootMount,
		MountPath: "/mnt/root",
	}
}

func prepareVolumes(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(instance))
	}

	if fs.ReadOnly.Enabled {
		volumes = append(volumes, getInstallationVolume(fs))
	}

	return volumes
}

func getInstallationVolume(fs *dynatracev1alpha1.FullStackSpec) corev1.Volume {
	return corev1.Volume{
		Name:         oneagentInstallationMountName,
		VolumeSource: fs.ReadOnly.GetInstallationVolume(),
	}
}

func getCertificateVolume(instance *dynatracev1alpha1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: "certs",
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

func getRootVolume() corev1.Volume {
	return corev1.Volume{
		Name: hostRootMount,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/",
			},
		},
	}
}
