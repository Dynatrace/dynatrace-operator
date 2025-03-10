package oneagent

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common"
	corev1 "k8s.io/api/core/v1"
)

const (
	binSubPath       = "bin"
	binInitMountPath = "/mnt/bin"

	ldPreloadPath    = "/etc/ld.so.preload"
	ldPreloadSubPath = "oneagent/ld.so.preload" // TODO: Get from the bootsrapper lib.
)

func addVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      common.ConfigVolumeName,
			MountPath: installPath,
			SubPath:   binSubPath,
		},
		corev1.VolumeMount{
			Name:      common.ConfigVolumeName,
			MountPath: ldPreloadPath,
			SubPath:   filepath.Join(common.InitConfigSubPath, ldPreloadSubPath),
		},
	)

	common.AddConfigVolumeMount(container)
}

func addInitVolumeMounts(initContainer *corev1.Container) {
	initContainer.VolumeMounts = append(initContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      common.ConfigVolumeName,
			MountPath: binInitMountPath,
			SubPath:   binSubPath,
		},
	)

	common.AddInitConfigVolumeMount(initContainer)
	common.AddInitInputVolumeMount(initContainer)
}
