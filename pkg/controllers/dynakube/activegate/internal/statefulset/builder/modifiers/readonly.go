package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

var _ volumeModifier = ReadOnlyModifier{}
var _ volumeMountModifier = ReadOnlyModifier{}
var _ builder.Modifier = ReadOnlyModifier{}

func NewReadOnlyModifier(dk dynakube.DynaKube) ReadOnlyModifier {
	return ReadOnlyModifier{
		dk: dk,
	}
}

type ReadOnlyModifier struct {
	presentVolumes []corev1.Volume
	presentMounts  []corev1.VolumeMount
	dk             dynakube.DynaKube
}

func (mod ReadOnlyModifier) Enabled() bool {
	return true // TODO: Investigate moving this package to the default statefulset
}

func (mod ReadOnlyModifier) Modify(sts *appsv1.StatefulSet) error {
	mod.presentVolumes = sts.Spec.Template.Spec.Volumes
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)

	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.SecurityContext.ReadOnlyRootFilesystem = ptr.To(true)
	mod.presentMounts = baseContainer.VolumeMounts
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod ReadOnlyModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
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
			Name: consts.GatewayConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (mod ReadOnlyModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
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
			Name:      consts.GatewayConfigVolumeName,
			MountPath: consts.GatewayConfigMountPoint,
		},
	}
}
