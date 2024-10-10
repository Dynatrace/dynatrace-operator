package oneagent

import (
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	corev1 "k8s.io/api/core/v1"
)

func (mut *Mutator) addVolumes(pod *corev1.Pod, dk dynakube.DynaKube) {
	addInjectionConfigVolume(pod)
	addOneAgentVolumes(pod, dk)

	if dk.FeatureReadOnlyCsiVolume() {
		addVolumesForReadOnlyCSI(pod)
	}
}

func addOneAgentVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      oneAgentShareVolumeName,
			MountPath: preloadPath,
			SubPath:   consts.LdPreloadFilename,
		},
		corev1.VolumeMount{
			Name:      OneAgentBinVolumeName,
			MountPath: installPath,
		},
		corev1.VolumeMount{
			Name:      oneAgentShareVolumeName,
			MountPath: containerConfPath,
			SubPath:   getContainerConfSubPath(container.Name),
		},
	)
}

func addVolumeMountsForReadOnlyCSI(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      oneagentConfVolumeName,
			MountPath: OneAgentConfMountPath,
		},
		corev1.VolumeMount{
			Name:      oneagentDataStorageVolumeName,
			MountPath: oneagentDataStorageMountPath,
		},
		corev1.VolumeMount{
			Name:      oneagentLogVolumeName,
			MountPath: oneagentLogMountPath,
		},
	)
}

func getContainerConfSubPath(containerName string) string {
	return fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, containerName)
}

func addCertVolumeMounts(container *corev1.Container, dk dynakube.DynaKube) {
	if dk.ActiveGate().HasCaCert() || dk.Spec.TrustedCAs != "" {
		container.VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      oneAgentShareVolumeName,
				MountPath: filepath.Join(oneAgentCustomKeysPath, consts.CustomCertsFileName),
				SubPath:   consts.CustomCertsFileName,
			})
	}

	if dk.Spec.TrustedCAs != "" {
		container.VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      oneAgentShareVolumeName,
				MountPath: filepath.Join(oneAgentCustomKeysPath, consts.CustomProxyCertsFileName),
				SubPath:   consts.CustomProxyCertsFileName,
			})
	}
}

func addInitVolumeMounts(initContainer *corev1.Container, dk dynakube.DynaKube) {
	volumeMounts := []corev1.VolumeMount{
		{Name: OneAgentBinVolumeName, MountPath: consts.AgentBinDirMount},
		{Name: oneAgentShareVolumeName, MountPath: consts.AgentShareDirMount},
	}
	if dk.FeatureReadOnlyCsiVolume() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: oneagentConfVolumeName, MountPath: consts.AgentConfInitDirMount})
	}

	initContainer.VolumeMounts = append(initContainer.VolumeMounts, volumeMounts...)
}

func addCurlOptionsVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      oneAgentShareVolumeName,
		MountPath: filepath.Join(oneAgentCustomKeysPath, consts.AgentCurlOptionsFileName),
		SubPath:   consts.AgentCurlOptionsFileName,
	})
}

func addInjectionConfigVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: injectionConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: consts.AgentInitSecretName,
				},
			},
		},
	)
}

func addInjectionConfigVolumeMount(container *corev1.Container) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{Name: injectionConfigVolumeName, MountPath: consts.AgentConfigDirMount},
	)
}

func addOneAgentVolumes(pod *corev1.Pod, dk dynakube.DynaKube) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name:         OneAgentBinVolumeName,
			VolumeSource: getInstallerVolumeSource(dk),
		},
		corev1.Volume{
			Name: oneAgentShareVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func addVolumesForReadOnlyCSI(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: oneagentConfVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: oneagentDataStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: oneagentLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func getInstallerVolumeSource(dk dynakube.DynaKube) corev1.VolumeSource {
	volumeSource := corev1.VolumeSource{}
	if dk.NeedsCSIDriver() {
		volumeSource.CSI = &corev1.CSIVolumeSource{
			Driver:   dtcsi.DriverName,
			ReadOnly: address.Of(dk.FeatureReadOnlyCsiVolume()),
			VolumeAttributes: map[string]string{
				csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
				csivolumes.CSIVolumeAttributeDynakubeField: dk.Name,
				csivolumes.CSIVolumeAttributeRetryTimeout:  dk.FeatureMaxCSIRetryTimeout().String(),
			},
		}
	} else {
		volumeSource.EmptyDir = &corev1.EmptyDirVolumeSource{}
	}

	return volumeSource
}
