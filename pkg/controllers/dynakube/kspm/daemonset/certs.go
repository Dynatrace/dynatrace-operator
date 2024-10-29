package daemonset

import (
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	corev1 "k8s.io/api/core/v1"
)

const (
	certVolumeName = "certs"
	certFileName   = "ca-chain.pem"
	certFolderPath = "/var/lib/dynatrace/ncc/customkeys"
	certFileEnv    = "DT_CA_CERTIFICATE_FILE"
)

func getCertVolume(dk dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: certVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.ActiveGate().TlsSecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  dynakube.TLSCertKey,
						Path: certFileName,
					},
				},
			},
		},
	}
}

func getCertMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      certVolumeName,
		MountPath: certFolderPath,
		SubPath:   certFileName,
	}
}

func getCertEnv() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  certFileEnv,
		Value: path.Join(certFolderPath, certFileName),
	}
}
