package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = ReadOnlyModifier{}
var _ volumeMountModifier = ReadOnlyModifier{}
var _ builder.Modifier = ReadOnlyModifier{}

func NewReadOnlyModifier(dynakube dynatracev1beta1.DynaKube) ReadOnlyModifier {
	return ReadOnlyModifier{
		dynakube: dynakube,
	}
}

type ReadOnlyModifier struct {
	dynakube       dynatracev1beta1.DynaKube
	presentVolumes []corev1.Volume
	presentMounts  []corev1.VolumeMount
}

func (mod ReadOnlyModifier) Enabled() bool {
	return mod.dynakube.FeatureActiveGateReadOnlyFilesystem()
}

func (mod ReadOnlyModifier) Modify(sts *appsv1.StatefulSet) {
	mod.presentVolumes = sts.Spec.Template.Spec.Volumes
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)

	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.SecurityContext.ReadOnlyRootFilesystem = address.Of(true)
	mod.presentMounts = baseContainer.VolumeMounts
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
}

func (mod ReadOnlyModifier) getVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: consts.GatewayLibTempVolumeName,
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
			Name: consts.GatewayLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: consts.GatewayTmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}}

	_, err := kubeobjects.GetVolumeByName(mod.presentVolumes, consts.GatewayConfigVolumeName)
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

func (mod ReadOnlyModifier) getVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.GatewayLibTempVolumeName,
			MountPath: consts.GatewayLibTempMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.GatewayDataVolumeName,
			MountPath: consts.GatewayDataMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.GatewayLogVolumeName,
			MountPath: consts.GatewayLogMountPoint,
		},
		{
			ReadOnly:  false,
			Name:      consts.GatewayTmpVolumeName,
			MountPath: consts.GatewayTmpMountPoint,
		}}

	neededMount := corev1.VolumeMount{
		ReadOnly:  false,
		Name:      consts.GatewayConfigVolumeName,
		MountPath: consts.GatewayConfigMountPoint,
	}
	if !kubeobjects.IsVolumeMountPresent(mod.presentMounts, neededMount) {
		volumeMounts = append(volumeMounts, neededMount)
	}
	return volumeMounts
}
