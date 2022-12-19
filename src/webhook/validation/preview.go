package validation

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	featurePreviewWarningMessage = `%s feature is in PREVIEW.`
	basePreviewWarning           = "PREVIEW features are NOT production ready and you may run into bugs."
)

func metricIngestPreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnOnCapabilityIfActive(dynatracev1beta1.MetricsIngestCapability.DisplayName, dynakube)
}

func warnOnCapabilityIfActive(capability dynatracev1beta1.CapabilityDisplayName, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsActiveGateMode(capability) {
		log.Info(fmt.Sprintf("DynaKube with %s was applied, warning was provided.", capability))
		return fmt.Sprintf(featurePreviewWarningMessage, capability)
	}
	return ""
}

func statsdIngestPreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnOnCapabilityIfActive(dynatracev1beta1.StatsdIngestCapability.DisplayName, dynakube)
}

func syntheticPreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	return warnOnCapabilityIfActive(dynatracev1beta1.SyntheticCapability.DisplayName, dynakube)
}
