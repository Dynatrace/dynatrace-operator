package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
)

func GetServiceFQDN(dk *dynakube.DynaKube) string {
	return capability.BuildServiceName(dk.Name) + "." + dk.Namespace
}
