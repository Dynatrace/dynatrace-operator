package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = ProxyModifier{}
var _ volumeMountModifier = ProxyModifier{}
var _ builder.Modifier = ProxyModifier{}

func NewProxyModifier(dynakube dynatracev1beta1.DynaKube) ProxyModifier {
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

func (mod ProxyModifier) Modify(sts *appsv1.StatefulSet) error {
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	return nil
}

func (mod ProxyModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.InternalProxySecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: capability.BuildProxySecretName(mod.dynakube.Name),
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
