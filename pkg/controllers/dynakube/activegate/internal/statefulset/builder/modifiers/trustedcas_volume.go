package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
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

func NewTrustedCAsVolumeModifier(dk dynakube.DynaKube) TrustedCAsModifier {
	return TrustedCAsModifier{
		dk: dk,
	}
}

type TrustedCAsModifier struct {
	dk dynakube.DynaKube
}

func (mod TrustedCAsModifier) Enabled() bool {
	return mod.dk.Spec.TrustedCAs != ""
}

func (mod TrustedCAsModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
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
						Name: mod.dk.Spec.TrustedCAs,
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
