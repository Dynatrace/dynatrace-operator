package dynakube

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

const oneagentEnableVolumeStorageEnvVarName = "ONEAGENT_ENABLE_VOLUME_STORAGE"
const oneagentInstallerScriptUrlEnvVarName = "ONEAGENT_INSTALLER_SCRIPT_URL"
const oneagentInstallerTokenEnvVarName = "ONEAGENT_INSTALLER_TOKEN"

var log = logger.Get().WithName("dynakube-validation")

type validator func(ctx context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string

var validators = []validator{
	NoApiUrl,
	IsInvalidApiUrl,
	IsThirdGenAPIUrl,
	missingCSIDaemonSet,
	disabledCSIForReadonlyCSIVolume,
	conflictingActiveGateConfiguration,
	invalidActiveGateCapabilities,
	duplicateActiveGateCapabilities,
	invalidActiveGateProxyUrl,
	conflictingOneAgentConfiguration,
	conflictingNodeSelector,
	conflictingNamespaceSelector,
	noResourcesAvailable,
	imageFieldSetWithoutCSIFlag,
	conflictingOneAgentVolumeStorageSettings,
	nameViolatesDNS1035,
	nameTooLong,
	namespaceSelectorViolateLabelSpec,
}

var warnings = []validator{
	missingActiveGateMemoryLimit,
	deprecatedFeatureFlagDisableActiveGateUpdates,
	deprecatedFeatureFlagDisableMetadataEnrichment,
	deprecatedFeatureFlagWillBeDeleted,
	deprecatedFeatureFlagMovedCRDField,
	unsupportedOneAgentImage,
	conflictingHostGroupSettings,
}

func SetLogger(logger logger.DtLogger) {
	log = logger
}
