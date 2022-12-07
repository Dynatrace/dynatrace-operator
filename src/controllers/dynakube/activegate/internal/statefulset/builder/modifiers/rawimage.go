package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
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

func (mod RawImageModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	envs, err := mod.getEnvs()
	if err != nil {
		return err
	}
	baseContainer.Env = append(baseContainer.Env, envs...)
	return nil
}

func (mod RawImageModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: connectioninfo.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dynakube.ActivegateTenantSecret(),
				},
			},
		},
	}
}

func (mod RawImageModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      connectioninfo.TenantSecretVolumeName,
			ReadOnly:  true,
			MountPath: connectioninfo.TenantTokenMountPoint,
			SubPath:   connectioninfo.TenantTokenName,
		},
	}
}

func (mod RawImageModifier) getEnvs() ([]corev1.EnvVar, error) {
	envVar, err := mod.tenantUuidEnvVar()
	if err != nil {
		return nil, err
	}
	return []corev1.EnvVar{envVar, mod.communicationEndpointEnvVar()}, nil
}

func (mod RawImageModifier) tenantUuidEnvVar() (corev1.EnvVar, error) {
	uuid, err := mod.dynakube.TenantUUID()
	if err != nil {
		return corev1.EnvVar{}, err
	}
	return corev1.EnvVar{
		Name:  consts.EnvDtTenant,
		Value: uuid,
	}, nil
}

func (mod RawImageModifier) communicationEndpointEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: consts.EnvDtServer,
		ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mod.dynakube.ActiveGateConnectionInfoConfigMapName(),
			},
			Key:      connectioninfo.CommunicationEndpointsName,
			Optional: address.Of[bool](false),
		}},
	}
}
