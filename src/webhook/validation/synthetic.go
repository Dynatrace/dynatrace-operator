package validation

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	errorInvalidSyntheticNodeType = `The DynaKube's specification requires illegally the synthetic node type: %v.
Make sure such a node is valid.
`
	errorInvalidSyntheticAutoscalerReplicaBounds = `The DynaKube's specification requires non-ascending replica limits.
Make sure such limits are valid.
`
	errorInvalidDynaMetricsToken = `The DynaKube's specification undefines DynaMetrics token.
Make sure such a token is valid.
`
)

func invalidSyntheticNodeType(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	isTypeValid := func() bool {
		switch dynakube.FeatureSyntheticNodeType() {
		case dynatracev1beta1.SyntheticNodeXs,
			dynatracev1beta1.SyntheticNodeS,
			dynatracev1beta1.SyntheticNodeM:
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

func invalidSyntheticAutoscalerReplicaBounds(validator *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsSyntheticMonitoringEnabled() &&
		dynakube.FeatureSyntheticAutoscalerMinReplicas() >= dynakube.FeatureSyntheticAutoscalerMaxReplicas() {
		log.Info(
			"requested dynakube has the invalid replica limits",
			"name", dynakube.Name,
			"namespace", dynakube.Namespace)
		return errorInvalidSyntheticAutoscalerReplicaBounds
	}
	return ""
}

func invalidDynaMetricsToken(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	isTokenless := func() bool {
		return len(strings.TrimSpace(dynakube.FeatureDynaMetricsToken())) == 0
	}

	if dynakube.IsSyntheticMonitoringEnabled() && isTokenless() {
		log.Info(
			"requested dynakube has the void DynaMetrics token",
			"name", dynakube.Name,
			"namespace", dynakube.Namespace)
		return errorInvalidDynaMetricsToken
	}
	return ""
}
