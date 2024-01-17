package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = TrustedCAsModifier{}
var _ volumeMountModifier = TrustedCAsModifier{}
var _ builder.Modifier = TrustedCAsModifier{}

const (
	volumeName     = "trustedcas"
	trustedCAsDir  = "/var/lib/dynatrace/secrets/rootca"
	trustedCAsFile = "rootca.pem"
)

func NewTrustedCAsModifier(dynakube dynatracev1beta1.DynaKube) TrustedCAsModifier {
	return TrustedCAsModifier{
		dynakube: dynakube,
	}
}

type TrustedCAsModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod TrustedCAsModifier) Enabled() bool {
	return mod.dynakube.Spec.TrustedCAs != ""
}

func (mod TrustedCAsModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	return nil
}

func (mod TrustedCAsModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mod.dynakube.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: trustedCAsFile,
						},
					},
				},
			},
		},
	}
}

func (mod TrustedCAsModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      volumeName,
			MountPath: trustedCAsDir,
		},
	}
}
