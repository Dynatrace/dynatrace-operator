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
	if dynakube.IsActiveGateMode(dynatracev1beta1.MetricsIngestCapability.DisplayName) {
		log.Info("DynaKube with metrics-ingest was applied, warning was provided.")
		return fmt.Sprintf(featurePreviewWarningMessage, "metrics-ingest")
	}
	return ""
}

func statsdIngestPreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsActiveGateMode(dynatracev1beta1.StatsdIngestCapability.DisplayName) {
		log.Info("DynaKube with statsd-ingest was applied, warning was provided.")
		return fmt.Sprintf(featurePreviewWarningMessage, "statsd-ingest")
	}
	return ""
}
