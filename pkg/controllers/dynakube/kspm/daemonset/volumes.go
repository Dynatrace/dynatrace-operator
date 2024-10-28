package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	corev1 "k8s.io/api/core/v1"
)

const (
	tokenVolumeName           = "kspm-token"
	tokenMountPath            = "/var/lib/dynatrace/secrets/tokens/kspm/node-configuration-collector"
	tokenSecretHashAnnotation = api.InternalFlagPrefix + "kspm-token-secret-hash"

	nodeRootVolumeName = "node-root"
	nodeRootMountPath  = "/node_root"
)

func getVolumes(dk dynakube.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{
		getNodeRootVolume(),
		getTokenVolume(dk),
	}

	if needsCerts(dk) {
		volumes = append(volumes, getCertVolume(dk))
	}

	return volumes
}

func getMounts(dk dynakube.DynaKube) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		getNodeRootVolumeMount(),
		getTokenVolumeMount(),
	}

	if needsCerts(dk) {
		mounts = append(mounts, getCertMount())
	}

	return mounts
}

func getTokenVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      tokenVolumeName,
		MountPath: tokenMountPath,
		SubPath:   kspm.TokenSecretKey, // TODO: is this correct?
	}
}

func getTokenVolume(dk dynakube.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: tokenVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dk.KSPM().GetTokenSecretName(),
			},
		},
	}
}

func getNodeRootVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      nodeRootVolumeName,
		MountPath: nodeRootMountPath,
		ReadOnly:  true,
	}
}

func getNodeRootVolume() corev1.Volume {
	return corev1.Volume{
		Name: nodeRootVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/",
			},
		},
	}
}
