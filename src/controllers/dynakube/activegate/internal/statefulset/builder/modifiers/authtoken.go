package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/util/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = AuthTokenModifier{}
var _ volumeMountModifier = AuthTokenModifier{}
var _ builder.Modifier = AuthTokenModifier{}

func NewAuthTokenModifier(dynakube dynatracev1beta1.DynaKube) AuthTokenModifier {
	return AuthTokenModifier{
		dynakube: dynakube,
	}
}

type AuthTokenModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod AuthTokenModifier) Enabled() bool {
	return mod.dynakube.UseActiveGateAuthToken()
}

func (mod AuthTokenModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
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
					SecretName: mod.dynakube.ActiveGateAuthTokenSecret(),
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
