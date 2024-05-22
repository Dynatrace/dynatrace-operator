package modifiers

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	corev1 "k8s.io/api/core/v1"
)

const modifierEnvPriority = prioritymap.MediumPriority

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

func GenerateAllModifiers(dynakube dynatracev1beta2.DynaKube, capability capability.Capability, agBaseContainerEnvMap *prioritymap.Map) []builder.Modifier {
	return []builder.Modifier{
		NewAuthTokenModifier(dynakube),
		NewSSLVolumeModifier(dynakube),
		NewCertificatesModifier(dynakube),
		NewTrustedCAsVolumeModifier(dynakube),
		NewCustomPropertiesModifier(dynakube, capability),
		NewProxyModifier(dynakube),
		NewRawImageModifier(dynakube, agBaseContainerEnvMap),
		NewReadOnlyModifier(dynakube),
		NewServicePortModifier(dynakube, capability, agBaseContainerEnvMap),
		NewKubernetesMonitoringModifier(dynakube, capability),
	}
}
