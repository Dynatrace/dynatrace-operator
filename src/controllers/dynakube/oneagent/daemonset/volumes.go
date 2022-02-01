package daemonset

import (
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const OneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"

func prepareVolumeMounts(instance *dynatracev1beta1.DynaKube) []corev1.VolumeMount {
	rootMount := getRootMount()
	var volumeMounts []corev1.VolumeMount

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getCertificateMount())
	}

	if instance.HasActiveGateTLS() {
		volumeMounts = append(volumeMounts, getTLSMount())
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

func getTLSMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "tls",
		MountPath: filepath.Join("/mnt/root", OneAgentCustomKeysPath),
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

	if instance.HasActiveGateTLS() {
		volumes = append(volumes, getTLSVolume(instance))
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

func getTLSVolume(instance *dynatracev1beta1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: "tls",
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
