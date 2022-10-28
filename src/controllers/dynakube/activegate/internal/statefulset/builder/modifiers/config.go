package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
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

var (
	log = logger.Factory.GetLogger("activegate-statefulset-builder")
)

func GenerateAllModifiers(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) []builder.Modifier {
	return []builder.Modifier{
		NewKubernetesMonitoringModifier(dynakube, capability),
		NewServicePortModifier(dynakube, capability),
		NewAuthTokenModifier(dynakube),
		NewCertificatesModifier(dynakube),
		NewCustomPropertiesModifier(dynakube, capability),
		NewExtensionControllerModifier(dynakube, capability),
		NewProxyModifier(dynakube),
		NewRawImageModifier(dynakube),
		NewReadOnlyModifier(dynakube),
		NewStatsdModifier(dynakube, capability),
		newSyntheticModifier(dynakube),
	}
}
