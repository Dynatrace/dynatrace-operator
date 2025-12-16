package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	corev1 "k8s.io/api/core/v1"
)

const (
	certVolumeName = "certs"
	certFolderPath = "/var/lib/dynatrace/ncc/customkeys/ca-chain.pem"
	certFileEnv    = "DT_CA_CERTIFICATE_FILE"
)

func getCertVolume(dk dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: certVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.ActiveGate().GetTLSSecretName(),
			},
		},
	}
}

func getCertMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      certVolumeName,
		MountPath: certFolderPath,
		SubPath:   dynakube.ServerCertKey,
	}
}

func getCertEnv() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  certFileEnv,
		Value: certFolderPath,
	}
}
