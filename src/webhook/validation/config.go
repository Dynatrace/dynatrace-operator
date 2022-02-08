package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var log = logger.NewDTLogger().WithName("validation-webhook")

type validator func(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string

var validators = []validator{
	noApiUrl,
	isInvalidApiUrl,
	missingCSIDaemonSet,
	conflictingActiveGateConfiguration,
	invalidActiveGateCapabilities,
	duplicateActiveGateCapabilities,
	conflictingOneAgentConfiguration,
	conflictingNodeSelector,
	conflictingNamespaceSelector,
}

var warnings = []validator{
	oneAgentModePreviewWarning,
	metricIngestPreviewWarning,
	statsdIngestPreviewWarning,
	missingActiveGateMemoryLimit,
}
