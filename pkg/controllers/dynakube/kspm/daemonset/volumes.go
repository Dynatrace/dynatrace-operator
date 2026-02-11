package daemonset

import (
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	tokenVolumeName           = "kspm-token"
	tokenMountPath            = "/var/lib/dynatrace/secrets/tokens/kspm/node-configuration-collector"
	tokenSecretHashAnnotation = api.InternalFlagPrefix + "kspm-token-secret-hash"

	nodeRootMountPath = "/node_root"
)

func getVolumes(dk dynakube.DynaKube) []corev1.Volume {
	var volumes []corev1.Volume

	volumes = append(volumes, getNodeVolumes(dk.KSPM().GetUniqueMappedHostPaths())...)
	volumes = append(volumes, getTokenVolume(dk))

	if dk.ActiveGate().HasCaCert() {
		volumes = append(volumes, getCertVolume(dk))
	}

	return volumes
}

func getMounts(dk dynakube.DynaKube) []corev1.VolumeMount {
	var mounts []corev1.VolumeMount

	mounts = append(mounts, getNodeVolumeMounts(dk.KSPM().GetUniqueMappedHostPaths())...)
	mounts = append(mounts, getTokenVolumeMount())

	if dk.ActiveGate().HasCaCert() {
		mounts = append(mounts, getCertMount())
	}

	return mounts
}

func getTokenVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      tokenVolumeName,
		MountPath: tokenMountPath,
		SubPath:   kspm.TokenSecretKey,
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

func getNodeVolumeMounts(mappedHostPaths []string) []corev1.VolumeMount {
	volumeMounts := make([]corev1.VolumeMount, len(mappedHostPaths))
	for i, path := range mappedHostPaths {
		volumeMounts[i] = corev1.VolumeMount{
			Name:      getVolumeName(i + 1),
			MountPath: filepath.Join(nodeRootMountPath, path),
			ReadOnly:  true,
		}
	}

	return volumeMounts
}

func getNodeVolumes(mappedHostPaths []string) []corev1.Volume {
	volumes := make([]corev1.Volume, len(mappedHostPaths))
	for i, path := range mappedHostPaths {
		volumes[i] = corev1.Volume{
			Name: getVolumeName(i + 1),
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: path,
					Type: ptr.To(corev1.HostPathDirectory),
				},
			},
		}
	}

	return volumes
}

func getVolumeName(i int) string {
	return fmt.Sprintf("node-path-%d", i)
}
