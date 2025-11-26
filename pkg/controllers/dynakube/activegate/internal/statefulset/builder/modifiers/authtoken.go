package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = AuthTokenModifier{}
var _ volumeMountModifier = AuthTokenModifier{}
var _ builder.Modifier = AuthTokenModifier{}

func NewAuthTokenModifier(dk dynakube.DynaKube) AuthTokenModifier {
	return AuthTokenModifier{
		dk: dk,
	}
}

type AuthTokenModifier struct {
	dk dynakube.DynaKube
}

func (mod AuthTokenModifier) Enabled() bool {
	return true // TODO: Investigate moving this package to the default statefulset
}

func (mod AuthTokenModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod AuthTokenModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.AuthTokenSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dk.ActiveGate().GetAuthTokenSecretName(),
				},
			},
		},
	}
}

func (mod AuthTokenModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      consts.AuthTokenSecretVolumeName,
			ReadOnly:  true,
			MountPath: consts.AuthTokenMountPoint,
			SubPath:   authtoken.ActiveGateAuthTokenName,
		},
	}
}
