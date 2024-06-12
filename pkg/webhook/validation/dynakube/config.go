package dynakube

import (
	"context"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const oneagentEnableVolumeStorageEnvVarName = "ONEAGENT_ENABLE_VOLUME_STORAGE"
const oneagentInstallerScriptUrlEnvVarName = "ONEAGENT_INSTALLER_SCRIPT_URL"
const oneagentInstallerTokenEnvVarName = "ONEAGENT_INSTALLER_TOKEN"

var log = logd.Get().WithName("dynakube-validation")

type validator func(ctx context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta2.DynaKube) string

var validators = []validator{
	NoApiUrl,
	IsInvalidApiUrl,
	IsThirdGenAPIUrl,
	missingCSIDaemonSet,
	disabledCSIForReadonlyCSIVolume,
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
	imageFieldHasTenantImage,
}

var warnings = []validator{
	missingActiveGateMemoryLimit,
	deprecatedFeatureFlagDisableActiveGateUpdates,
	deprecatedFeatureFlagWillBeDeleted,
	deprecatedFeatureFlagMovedCRDField,
	unsupportedOneAgentImage,
	conflictingHostGroupSettings,
}

func SetLogger(logger logd.Logger) {
	log = logger
}
