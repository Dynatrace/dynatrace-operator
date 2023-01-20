package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	corev1 "k8s.io/api/core/v1"
)

type volumeModifier interface {
	getVolumes() []corev1.Volume
}

type volumeMountModifier interface {
	getVolumeMounts() []corev1.VolumeMount
}

type envModifier interface {
	getEnvs() []corev1.EnvVar
}

type initContainerModifier interface {
	getInitContainers() []corev1.Container
}

func GenerateAllModifiers(dynaKube dynatracev1beta1.DynaKube, capability capability.Capability) []builder.Modifier {
	generated := []builder.Modifier{
		NewAuthTokenModifier(dynaKube),
		NewCertificatesModifier(dynaKube),
		NewCustomPropertiesModifier(dynaKube, capability),
		NewProxyModifier(dynaKube),
		NewRawImageModifier(dynaKube),
		NewReadOnlyModifier(dynaKube),
	}

	if capability.AssistsSynthetic() {
		generated = append(generated, newSyntheticModifier(dynaKube))
	} else {
		generated = append(
			generated,
			NewKubernetesMonitoringModifier(dynaKube, capability),
			NewServicePortModifier(dynaKube, capability))
	}

	return generated
}
