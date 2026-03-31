package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	eecconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

var _ volumeModifier = EECModifier{}
var _ volumeMountModifier = EECModifier{}
var _ builder.Modifier = EECModifier{}

const (
	eecVolumeName = "eec-token"
	eecMountPath  = "/var/lib/dynatrace/secrets/eec/token"
	eecFile       = "eec.token"
)

func NewEECVolumeModifier(dk dynakube.DynaKube) EECModifier {
	return EECModifier{
		dk: dk,
	}
}

type EECModifier struct {
	dk dynakube.DynaKube
}

func (mod EECModifier) Enabled() bool {
	return mod.dk.Extensions().IsAnyEnabled()
}

func (mod EECModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod EECModifier) getVolumes() []corev1.Volume {
	mode := ptr.To(int32(0o640))

	return []corev1.Volume{
		{
			Name: eecVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  mod.dk.Extensions().GetTokenSecretName(),
					DefaultMode: mode,
					Items: []corev1.KeyToPath{
						{
							Key:  eecconsts.TokenSecretKey,
							Path: eecFile,
							Mode: mode,
						},
					},
				},
			},
		},
	}
}

func (mod EECModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      eecVolumeName,
			MountPath: eecMountPath,
		},
	}
}
