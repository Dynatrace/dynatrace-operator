package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/go-logr/logr"
)

const oneagentEnableVolumeStorageEnvVarName = "ONEAGENT_ENABLE_VOLUME_STORAGE"

var log = logger.Factory.GetLogger("validation")

type validator func(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string

var validators = []validator{
	NoApiUrl,
	IsInvalidApiUrl,
	missingCSIDaemonSet,
	disabledCSIForReadonlyCSIVolume,
	conflictingActiveGateConfiguration,
	exclusiveSyntheticCapability,
	invalidActiveGateCapabilities,
	duplicateActiveGateCapabilities,
	invalidActiveGateProxyUrl,
	conflictingOneAgentConfiguration,
	conflictingNodeSelector,
	conflictingNamespaceSelector,
	conflictingReadOnlyFilesystemAndMultipleOsAgentsOnNode,
	noResourcesAvailable,
	imageFieldSetWithoutCSIFlag,
	conflictingOneAgentVolumeStorageSettings,
	invalidSyntheticNodeType,
}

var warnings = []validator{
	deprecatedFeatureFlagFormat,
	metricIngestPreviewWarning,
	missingActiveGateMemoryLimit,
	deprecatedFeatureFlagDisableActiveGateUpdates,
	deprecatedFeatureFlagDisableActiveGateRawImage,
	deprecatedFeatureFlagDisableHostsRequests,
	deprecatedFeatureFlagDisableReadOnlyAgent,
	deprecatedFeatureFlagDisableWebhookReinvocationPolicy,
	deprecatedFeatureFlagDisableMetadataEnrichment,
	ineffectiveReadOnlyHostFsFeatureFlag,
	syntheticPreviewWarning,
}

func SetLogger(logger logr.Logger) {
	log = logger
}
