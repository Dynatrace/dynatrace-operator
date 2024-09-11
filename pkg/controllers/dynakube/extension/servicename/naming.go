package servicename

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
)

func BuildPortName() string {
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsCollectorTargetPortName
}

func BuildFQDN(dk *dynakube.DynaKube) string {
	return Build(dk) + "." + dk.Namespace
}

func Build(dk *dynakube.DynaKube) string {
	return dk.Name + consts.ExtensionsControllerSuffix
}
