package daemonset

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func prepareVolumeMounts(instance *dynatracev1.DynaKube) []corev1.VolumeMount {
	rootMount := getRootMount()
	var volumeMounts []corev1.VolumeMount

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getCertificateMount())
	}

	volumeMounts = append(volumeMounts, rootMount)
	return volumeMounts
}

func (dsInfo *HostMonitoring) appendReadOnlyVolume(daemonset *appsv1.DaemonSet) {
	if dsInfo.instance.ReadOnly() {
		daemonset.Spec.Template.Spec.Volumes = append(daemonset.Spec.Template.Spec.Volumes, getReadOnlyVolume(dsInfo.instance))
	}
}

func getReadOnlyVolume(dk *dynatracev1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: oneagentInstallationMountName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func (dsInfo *HostMonitoring) appendReadOnlyVolumeMount(daemonset *appsv1.DaemonSet) {
	if dsInfo.instance.ReadOnly() {
		daemonset.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			daemonset.Spec.Template.Spec.Containers[0].VolumeMounts,
			getInstallationMount())
	}
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

func (dsInfo *HostMonitoring) setRootMountReadability(result *appsv1.DaemonSet) {
	volumeMounts := result.Spec.Template.Spec.Containers[0].VolumeMounts
	for idx, mount := range volumeMounts {
		if mount.Name == hostRootMount {
			// using index here since range returns a copy not a reference
			volumeMounts[idx].ReadOnly = dsInfo.instance.ReadOnly()
		}
	}
}

func prepareVolumes(instance *dynatracev1.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(instance))
	}

	return volumes
}

func getCertificateVolume(instance *dynatracev1.DynaKube) corev1.Volume {
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
