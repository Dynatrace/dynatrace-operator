package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingOneagentMode = `The DynaKube specification attempts to use multiple OneAgent modes simultaneously, which is not supported.`

	errorImageFieldSetWithoutCSIFlag = `The DynaKube specification attempts to enable ApplicationMonitoring mode and retrieve the respective image, but the CSI driver is not enabled.`

	errorNodeSelectorConflict = `The Dynakube specification conflicts with another Dynakube's OneAgent or Standalone-LogMonitoring. Only one Agent per node is supported.
Use a nodeSelector to avoid this conflict. Conflicting DynaKubes: %s`

	errorVolumeStorageReadOnlyModeConflict = `The DynaKube specification specifies a read-only host file system while OneAgent has volume storage enabled.`

	errorPublicImageWithWrongConfig = `Custom OneAgent image is only supported when CSI Driver is used.`

	warningOneAgentInstallerEnvVars = `The environment variables ONEAGENT_INSTALLER_SCRIPT_URL and ONEAGENT_INSTALLER_TOKEN are only relevant for an unsupported image type. Please ensure you are using a supported image.`

	warningHostGroupConflict = `The DynaKube specification sets the host group using the --set-host-group parameter. Instead, specify the new spec.oneagent.hostGroup field. If both settings are used, the new field takes precedence over the parameter.`

	versionRegex = `^\d+.\d+.\d+.\d{8}-\d{6}$`

	versionInvalidMessage = "The OneAgent's version is only valid in the format 'major.minor.patch.timestamp', e.g. 1.0.0.20240101-000000"
)

func conflictingOneAgentConfiguration(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	counter := 0
	if dk.OneAgent().IsApplicationMonitoringMode() {
		counter += 1
	}

	if dk.OneAgent().IsCloudNativeFullstackMode() {
		counter += 1
	}

	if dk.OneAgent().IsClassicFullStackMode() {
		counter += 1
	}

	if dk.OneAgent().IsHostMonitoringMode() {
		counter += 1
	}

	if counter > 1 {
		log.Info("requested dynakube has conflicting one agent configuration", "name", dk.Name, "namespace", dk.Namespace)

		return errorConflictingOneagentMode
	}

	return ""
}

func conflictingOneAgentNodeSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsDaemonsetRequired() && !dk.LogMonitoring().IsStandalone() {
		return ""
	}

	validDynakubes := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, validDynakubes, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	oneAgentNodeSelector := dk.OneAgent().GetNodeSelector(dk.LogMonitoring().GetNodeSelector())
	conflictingDynakubes := make(map[string]bool)

	for _, item := range validDynakubes.Items {
		if item.Name == dk.Name {
			continue
		}

		if hasLogMonitoringSelectorConflict(dk, &item) || hasOneAgentSelectorConflict(dk, &item) {
			if hasConflictingMatchLabels(oneAgentNodeSelector, item.OneAgent().GetNodeSelector(dk.LogMonitoring().GetNodeSelector())) {
				log.Info("requested dynakube has conflicting OneAgent nodeSelector", "name", dk.Name, "namespace", dk.Namespace)

				conflictingDynakubes[item.Name] = true
			}
		}
	}

	if len(conflictingDynakubes) > 0 {
		return fmt.Sprintf(errorNodeSelectorConflict, mapKeysToString(conflictingDynakubes, ", "))
	}

	return ""
}

func hasLogMonitoringSelectorConflict(dk1, dk2 *dynakube.DynaKube) bool {
	return dk1.LogMonitoring().IsStandalone() && dk1.ApiUrl() == dk2.ApiUrl() &&
		(dk2.OneAgent().IsDaemonsetRequired() || dk2.LogMonitoring().IsStandalone()) &&
		hasConflictingMatchLabels(dk1.OneAgent().GetNodeSelector(dk1.LogMonitoring().GetNodeSelector()),
			dk2.OneAgent().GetNodeSelector(dk2.LogMonitoring().GetNodeSelector()))
}

func hasOneAgentSelectorConflict(dk1, dk2 *dynakube.DynaKube) bool {
	return dk1.OneAgent().IsDaemonsetRequired() &&
		(dk2.OneAgent().IsDaemonsetRequired() || dk2.LogMonitoring().IsStandalone() && dk1.ApiUrl() == dk2.ApiUrl()) &&
		hasConflictingMatchLabels(dk1.OneAgent().GetNodeSelector(dk1.LogMonitoring().GetNodeSelector()),
			dk2.OneAgent().GetNodeSelector(dk2.LogMonitoring().GetNodeSelector()))
}

func mapKeysToString(m map[string]bool, sep string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return strings.Join(keys, sep)
}

func publicImageSetWithoutReadOnlyMode(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if !dk.OneAgent().IsReadOnlyOneAgentsMode() && dk.OneAgent().GetCustomImage() != "" {
		return errorPublicImageWithWrongConfig
	}

	return ""
}

func imageFieldSetWithoutCSIFlag(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.OneAgent().IsApplicationMonitoringMode() {
		if len(dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage) > 0 && !v.modules.CSIDriver {
			return errorImageFieldSetWithoutCSIFlag
		}
	}

	return ""
}

func hasConflictingMatchLabels(labelMap, otherLabelMap map[string]string) bool {
	if labelMap == nil || otherLabelMap == nil {
		return true
	}

	labelSelector := labels.SelectorFromSet(labelMap)
	otherLabelSelector := labels.SelectorFromSet(otherLabelMap)
	labelSelectorLabels := labels.Set(labelMap)
	otherLabelSelectorLabels := labels.Set(otherLabelMap)

	return labelSelector.Matches(otherLabelSelectorLabels) || otherLabelSelector.Matches(labelSelectorLabels)
}

func hasOneAgentVolumeStorageEnabled(dk *dynakube.DynaKube) (isEnabled bool, isSet bool) {
	envVar := env.FindEnvVar(dk.OneAgent().GetEnvironment(), oneagentEnableVolumeStorageEnvVarName)
	isSet = envVar != nil
	isEnabled = isSet && envVar.Value == "true"

	return
}

func unsupportedOneAgentImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if env.FindEnvVar(dk.OneAgent().GetEnvironment(), oneagentInstallerScriptUrlEnvVarName) != nil ||
		env.FindEnvVar(dk.OneAgent().GetEnvironment(), oneagentInstallerTokenEnvVarName) != nil {
		return warningOneAgentInstallerEnvVars
	}

	return ""
}

func conflictingOneAgentVolumeStorageSettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	volumeStorageEnabled, volumeStorageSet := hasOneAgentVolumeStorageEnabled(dk)
	if dk.OneAgent().IsReadOnlyOneAgentsMode() && volumeStorageSet && !volumeStorageEnabled {
		return errorVolumeStorageReadOnlyModeConflict
	}

	return ""
}

func conflictingHostGroupSettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.OneAgent().GetHostGroupAsParam() != "" {
		return warningHostGroupConflict
	}

	return ""
}

func isOneAgentVersionValid(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	agentVersion := dk.OneAgent().GetCustomVersion()
	if agentVersion == "" {
		return ""
	}

	_, err := dtversion.ToSemver(agentVersion)
	if err != nil {
		return versionInvalidMessage
	}

	match, err := regexp.MatchString(versionRegex, agentVersion)
	if err != nil || !match {
		return versionInvalidMessage
	}

	return ""
}
