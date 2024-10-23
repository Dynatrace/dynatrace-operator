package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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

func GenerateAllModifiers(dk dynakube.DynaKube, capability capability.Capability, agBaseContainerEnvMap *prioritymap.Map) []builder.Modifier {
	return []builder.Modifier{
		NewAuthTokenModifier(dk),
		NewSSLVolumeModifier(dk),
		NewCertificatesModifier(dk),
		NewTrustedCAsVolumeModifier(dk),
		NewCustomPropertiesModifier(dk, capability),
		NewProxyModifier(dk),
		NewRawImageModifier(dk, agBaseContainerEnvMap),
		NewReadOnlyModifier(dk),
		NewServicePortModifier(dk, capability, agBaseContainerEnvMap),
		NewKubernetesMonitoringModifier(dk, capability),
		NewEecVolumeModifier(dk),
		NewKspmModifier(dk),
	}
}
