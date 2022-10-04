package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/secret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ envModifier = RawImageModifier{}
var _ volumeModifier = RawImageModifier{}
var _ volumeMountModifier = RawImageModifier{}
var _ builder.Modifier = RawImageModifier{}

func NewRawImageModifier(dynakube dynatracev1beta1.DynaKube) RawImageModifier {
	return RawImageModifier{
		dynakube: dynakube,
	}
}

type RawImageModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod RawImageModifier) Enabled() bool {
	return !mod.dynakube.FeatureDisableActivegateRawImage()
}

func (mod RawImageModifier) Modify(sts *appsv1.StatefulSet) {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	baseContainer.Env = append(baseContainer.Env, mod.getEnvs()...)
}

func (mod RawImageModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dynakube.AGTenantSecret(),
				},
			},
		},
	}
}

func (mod RawImageModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      consts.TenantSecretVolumeName,
			ReadOnly:  true,
			MountPath: consts.TenantTokenMountPoint,
			SubPath:   secret.TenantTokenName,
		},
	}
}

func (mod RawImageModifier) getEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{mod.tenantUUIDNameEnvVar(), mod.communicationEndpointEnvVar()}
}

func (mod RawImageModifier) tenantUUIDNameEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: consts.EnvDtTenant,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: mod.dynakube.AGTenantSecret(),
				},
				Key: secret.TenantUuidName,
			},
		},
	}
}

func (mod RawImageModifier) communicationEndpointEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: consts.EnvDtServer,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: mod.dynakube.AGTenantSecret(),
				},
				Key: secret.CommunicationEndpointsName,
			},
		},
	}
}
