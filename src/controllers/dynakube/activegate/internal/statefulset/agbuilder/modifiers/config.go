package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	agbuilderTypes "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"k8s.io/apimachinery/pkg/types"
)

var (
	log = logger.NewDTLogger().WithName("activegate-statefulset-builder")
)

func GetAllModifiers(kubeID types.UID, configHash string, dynakube dynatracev1beta1.DynaKube, capability capability.Capability) []agbuilderTypes.Modifier {
	return []agbuilderTypes.Modifier{
		NewBaseModifier(kubeID, configHash, dynakube, capability),
		NewAuthTokenModifier(dynakube),
		NewCertificatesModifier(dynakube),
		NewCustomPropertiesModifier(dynakube, capability),
		NewExtensionControllerModifier(dynakube, capability),
		NewProxyModifier(dynakube),
		NewRawImageModifier(dynakube),
		NewReadOnlyModifier(dynakube),
		NewStatsdModifier(dynakube, capability),
	}
}
