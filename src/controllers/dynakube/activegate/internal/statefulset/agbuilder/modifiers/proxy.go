package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	agbuilderTypes "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewProxyModifier(dynakube dynatracev1beta1.DynaKube) agbuilderTypes.Modifier {
	return ProxyModifier{
		dynakube: dynakube,
	}
}

type ProxyModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod ProxyModifier) Enabled() bool {
	return mod.dynakube.NeedsActiveGateProxy()
}

func (mod ProxyModifier) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

}

func (mod ProxyModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.InternalProxySecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: capability.BuildProxySecretName(),
				},
			},
		},
	}
}

func (mod ProxyModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      consts.InternalProxySecretVolumeName,
			MountPath: consts.InternalProxySecretHostMountPath,
			SubPath:   consts.InternalProxySecretHost,
		},
		{
			ReadOnly:  true,
			Name:      consts.InternalProxySecretVolumeName,
			MountPath: consts.InternalProxySecretPortMountPath,
			SubPath:   consts.InternalProxySecretPort,
		},
		{
			ReadOnly:  true,
			Name:      consts.InternalProxySecretVolumeName,
			MountPath: consts.InternalProxySecretUsernameMountPath,
			SubPath:   consts.InternalProxySecretUsername,
		},
		{
			ReadOnly:  true,
			Name:      consts.InternalProxySecretVolumeName,
			MountPath: consts.InternalProxySecretPasswordMountPath,
			SubPath:   consts.InternalProxySecretPassword,
		},
	}
}
