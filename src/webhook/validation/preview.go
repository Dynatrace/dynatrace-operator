package validation

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	featurePreviewWarningMessage = `%s feature is in PREVIEW.`
	basePreviewWarning           = "PREVIEW features are NOT production ready and you may run into bugs."
)

func oneAgentModePreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.CloudNativeFullstackMode() {
		log.Info("DynaKube with cloudNativeFullStack was applied, warning was provided.")
		return fmt.Sprintf(featurePreviewWarningMessage, "cloudNativeFullStack")
	} else if dynakube.ApplicationMonitoringMode() && dynakube.NeedsCSIDriver() {
		log.Info("DynaKube with applicationMonitoring was applied, warning was provided.")
		return fmt.Sprintf(featurePreviewWarningMessage, "applicationMonitoring")
	}
	return ""
}

func metricIngestPreviewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsActiveGateMode(dynatracev1beta1.MetricsIngestCapability.DisplayName) {
		log.Info("DynaKube with metrics-ingest was applied, warning was provided.")
		return fmt.Sprintf(featurePreviewWarningMessage, "metrics-ingest")
	}
	return ""
}
