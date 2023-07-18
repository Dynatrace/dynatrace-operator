package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	return nil
}

func (mod ProxyModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: config.ProxySecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: proxy.BuildProxySecretName(mod.dynakube.Name),
				},
			},
		},
	}
}

func (mod ProxyModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{proxy.PrepareVolumeMount()}
}
