package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = KspmModifier{}
var _ volumeMountModifier = KspmModifier{}
var _ builder.Modifier = KspmModifier{}

const (
	kspmTokenVolumeName           = "kspm-token"
	kspmTokenMountPath            = "/var/lib/dynatrace/secrets/tokens/kspm/node-configuration-collector"
	kspmTokenSecretHashAnnotation = api.InternalFlagPrefix + "kspm-token-secret-hash"
)

func NewKspmModifier(dk dynakube.DynaKube) KspmModifier {
	return KspmModifier{
		dk: dk,
	}
}

type KspmModifier struct {
	dk dynakube.DynaKube
}

func (mod KspmModifier) Enabled() bool {
	return mod.dk.KSPM().IsEnabled() && mod.dk.ActiveGate().IsKubernetesMonitoringEnabled()
}

func (mod KspmModifier) Modify(sts *appsv1.StatefulSet) error {
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = map[string]string{}
	}

	sts.Spec.Template.Annotations[kspmTokenSecretHashAnnotation] = mod.dk.KSPM().TokenSecretHash
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod KspmModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: kspmTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dk.KSPM().GetTokenSecretName(),
				},
			},
		},
	}
}

func (mod KspmModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      kspmTokenVolumeName,
			MountPath: kspmTokenMountPath,
			SubPath:   kspm.TokenSecretKey,
		},
	}
}
