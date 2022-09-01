package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	agbuilderTypes "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewReadOnlyModifier(dynakube dynatracev1beta1.DynaKube) agbuilderTypes.Modifier {
	return ReadOnlyModifier{
		dynakube: dynakube,
	}
}

type ReadOnlyModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod ReadOnlyModifier) Enabled() bool {
	return mod.dynakube.FeatureActiveGateReadOnlyFilesystem()
}

func (mod ReadOnlyModifier) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes(sts.Spec.Template.Spec.Volumes)...)

	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.SecurityContext.ReadOnlyRootFilesystem = address.Of(true)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts(baseContainer.VolumeMounts)...)
}

func (mod ReadOnlyModifier) getVolumes(presentVolumes []corev1.Volume) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: consts.GatewayTempVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: consts.GatewayDataVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: consts.LogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: consts.TmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}}

	_, err := kubeobjects.GetVolumeByName(presentVolumes, consts.GatewayConfigVolumeName)
	if err != nil {
		volumes = append(volumes,
			corev1.Volume{
				Name: consts.GatewayConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}
	return volumes
}

func (mod ReadOnlyModifier) getVolumeMounts(presentMounts []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.GatewayTempVolumeName,
			MountPath: consts.GatewayTempMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.GatewayDataVolumeName,
			MountPath: consts.GatewayDataMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.LogVolumeName,
			MountPath: consts.LogMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.TmpVolumeName,
			MountPath: consts.TmpMountPoint,
		}}

	_, err := kubeobjects.GetVolumeMountByName(presentMounts, consts.GatewayConfigVolumeName)
	if err != nil {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      consts.GatewayConfigVolumeName,
				MountPath: consts.GatewayConfigMountPoint,
			},
		)
	}
	return volumeMounts
}
