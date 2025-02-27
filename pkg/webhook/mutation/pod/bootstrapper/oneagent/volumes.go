package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

func (mut *Mutator) addVolumes(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: oneAgentCodeModulesVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: oneAgentCodeModulesConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func addVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      oneAgentCodeModulesVolumeName,
			MountPath: installPath,
		},
		corev1.VolumeMount{
			Name:      oneAgentCodeModulesConfigVolumeName,
			MountPath: oneAgentCodeModulesConfigMountPath,
		},
	)
}

func addInitVolumeMounts(initContainer *corev1.Container) {
	initContainer.VolumeMounts = append(initContainer.VolumeMounts,
		corev1.VolumeMount{Name: oneAgentCodeModulesVolumeName, MountPath: consts.AgentBinDirMount},
		corev1.VolumeMount{Name: oneAgentCodeModulesConfigVolumeName, MountPath: consts.AgentConfigDirMount},
	)
}
