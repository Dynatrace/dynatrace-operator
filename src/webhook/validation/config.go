package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/go-logr/logr"
)

var log = logger.NewDTLogger().WithName("validation-webhook")

type validator func(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string

var validators = []validator{
	NoApiUrl,
	IsInvalidApiUrl,
	missingCSIDaemonSet,
	conflictingActiveGateConfiguration,
	invalidActiveGateCapabilities,
	duplicateActiveGateCapabilities,
	invalidActiveGateProxyUrl,
	conflictingOneAgentConfiguration,
	conflictingNodeSelector,
	conflictingNamespaceSelector,
	conflictingReadOnlyFilesystemAndMultipleOsAgentsOnNode,
	noResourcesAvailable,
	imageFieldSetWithoutCSIFlag,
}

var warnings = []validator{
	deprecatedFeatureFlagFormat,
	metricIngestPreviewWarning,
	statsdIngestPreviewWarning,
	missingActiveGateMemoryLimit,
	deprecatedFeatureFlagDisableActiveGateUpdates,
	deprecatedFeatureFlagDisableActiveGateRawImage,
	deprecatedFeatureFlagEnableActiveGateAuthToken,
	deprecatedFeatureFlagDisableHostsRequests,
	deprecatedFeatureFlagDisableReadOnlyAgent,
	deprecatedFeatureFlagDisableWebhookReinvocationPolicy,
	deprecatedFeatureFlagDisableMetadataEnrichment,
}

func SetLogger(logger logr.Logger) {
	log = logger
}
