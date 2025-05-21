package oneagent

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/volumes"
	corev1 "k8s.io/api/core/v1"
)

const (
	binSubPath       = "bin"
	binInitMountPath = "/mnt/bin"

	ldPreloadPath    = "/etc/ld.so.preload"
	ldPreloadSubPath = preload.ConfigPath
)

func addVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      volumes.ConfigVolumeName,
			MountPath: installPath,
			SubPath:   binSubPath,
		},
		corev1.VolumeMount{
			Name:      volumes.ConfigVolumeName,
			MountPath: ldPreloadPath,
			SubPath:   filepath.Join(volumes.InitConfigSubPath, ldPreloadSubPath),
		},
	)

	volumes.AddConfigVolumeMount(container)
}

func addInitVolumeMounts(initContainer *corev1.Container) {
	initContainer.VolumeMounts = append(initContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      volumes.ConfigVolumeName,
			MountPath: binInitMountPath,
			SubPath:   binSubPath,
		},
	)
}
