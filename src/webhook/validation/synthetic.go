package validation

import (
	"fmt"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
)

const (
	errorInvalidSyntheticNodeType = `The DynaKube's specification requires illegally the synthetic node type: %v.
Make sure such a node is valid.
`
)

func invalidSyntheticNodeType(dv *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	isTypeValid := func() bool {
		switch dynakube.FeatureSyntheticNodeType() {
		case dynatracev1.SyntheticNodeXs,
			dynatracev1.SyntheticNodeS,
			dynatracev1.SyntheticNodeM:
			return true
		}
		return false
	}

	if dynakube.IsSyntheticMonitoringEnabled() && !isTypeValid() {
		log.Info(
			"requested dynakube has the invalid synthetic node type",
			"name", dynakube.Name,
			"namespace", dynakube.Namespace)
		return fmt.Sprintf(errorInvalidSyntheticNodeType, dynakube.FeatureSyntheticNodeType())
	}
	return ""
}
