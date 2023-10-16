package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/parametermap"
	corev1 "k8s.io/api/core/v1"
)

const modifierEnvPriority = 2

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

func GenerateAllModifiers(dynakube dynatracev1beta1.DynaKube, capability capability.Capability, agBaseContainerEnvMap *parametermap.Map) []builder.Modifier {
	return []builder.Modifier{
		NewAuthTokenModifier(dynakube),
		NewCertificatesModifier(dynakube),
		NewCustomPropertiesModifier(dynakube, capability),
		NewProxyModifier(dynakube),
		NewRawImageModifier(dynakube, agBaseContainerEnvMap),
		NewReadOnlyModifier(dynakube),
		newSyntheticModifier(dynakube, capability, agBaseContainerEnvMap),
		NewServicePortModifier(dynakube, capability, agBaseContainerEnvMap),
		NewKubernetesMonitoringModifier(dynakube, capability),
	}
}
