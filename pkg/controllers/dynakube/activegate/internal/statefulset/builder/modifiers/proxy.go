package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = ProxyModifier{}
var _ volumeMountModifier = ProxyModifier{}
var _ builder.Modifier = ProxyModifier{}

func NewProxyModifier(dk dynakube.DynaKube) ProxyModifier {
	return ProxyModifier{
		dk: dk,
	}
}

type ProxyModifier struct {
	dk dynakube.DynaKube
}

func (mod ProxyModifier) Enabled() bool {
	return mod.dk.NeedsActiveGateProxy()
}

func (mod ProxyModifier) Modify(sts *appsv1.StatefulSet) error {
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod ProxyModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: proxy.SecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: proxy.BuildSecretName(mod.dk.Name),
				},
			},
		},
	}
}

func (mod ProxyModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{proxy.BuildVolumeMount()}
}
