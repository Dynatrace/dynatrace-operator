package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func prepareVolumeMounts(instance *dynatracev1beta1.DynaKube) []corev1.VolumeMount {
	rootMount := getRootMount()
	var volumeMounts []corev1.VolumeMount

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getCertificateMount())
	}

	volumeMounts = append(volumeMounts, rootMount)
	return volumeMounts
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

func prepareVolumes(instance *dynatracev1beta1.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(instance))
	}

	return volumes
}

func getCertificateVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
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
