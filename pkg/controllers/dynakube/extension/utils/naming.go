package utils

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
)

func BuildPortsName() string {
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsCollectorTargetPortName
}

func BuildFQDName(dk *dynakube.DynaKube) string {
	return BuildServiceName(dk) + "." + dk.Namespace
}

func BuildServiceName(dk *dynakube.DynaKube) string {
	return dk.Name + consts.ExtensionsControllerSuffix
}
