package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ envModifier = RawImageModifier{}
var _ volumeModifier = RawImageModifier{}
var _ volumeMountModifier = RawImageModifier{}
var _ builder.Modifier = RawImageModifier{}

func NewRawImageModifier(dk dynakube.DynaKube, envMap *prioritymap.Map) RawImageModifier {
	return RawImageModifier{
		dk:     dk,
		envMap: envMap,
	}
}

type RawImageModifier struct {
	envMap *prioritymap.Map
	dk     dynakube.DynaKube
}

func (mod RawImageModifier) Enabled() bool {
	return true // TODO: Investigate moving this package to the default statefulset
}

func (mod RawImageModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	baseContainer.Env = mod.getEnvs()

	return nil
}

func (mod RawImageModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: connectioninfo.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mod.dk.ActiveGate().GetTenantSecretName(),
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
			SubPath:   connectioninfo.TenantTokenKey,
		},
	}
}

func (mod RawImageModifier) getEnvs() []corev1.EnvVar {
	prioritymap.Append(mod.envMap,
		[]corev1.EnvVar{mod.tenantUUIDEnvVar(), mod.communicationEndpointEnvVar()},
		prioritymap.WithPriority(modifierEnvPriority))

	return mod.envMap.AsEnvVars()
}

func (mod RawImageModifier) tenantUUIDEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: connectioninfo.EnvDtTenant,
		ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mod.dk.ActiveGate().GetConnectionInfoConfigMapName(),
			},
			Key:      connectioninfo.TenantUUIDKey,
			Optional: address.Of(false),
		}}}
}

func (mod RawImageModifier) communicationEndpointEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: connectioninfo.EnvDtServer,
		ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mod.dk.ActiveGate().GetConnectionInfoConfigMapName(),
			},
			Key:      connectioninfo.CommunicationEndpointsKey,
			Optional: address.Of(false),
		}},
	}
}
