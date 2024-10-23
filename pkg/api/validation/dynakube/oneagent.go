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

	errorNodeSelectorConflict = `The DynaKube specification attempts to deploy a OneAgent, which conflicts with another DynaKube's OneAgent/LogMonitoring. Only one Agent per node is supported.
Use a nodeSelector to avoid this conflict. Conflicting DynaKubes: %s`

	errorVolumeStorageReadOnlyModeConflict = `The DynaKube specification specifies a read-only host file system while OneAgent has volume storage enabled.`

	warningOneAgentInstallerEnvVars = `The environment variables ONEAGENT_INSTALLER_SCRIPT_URL and ONEAGENT_INSTALLER_TOKEN are only relevant for an unsupported image type. Please ensure you are using a supported image.`

	warningHostGroupConflict = `The DynaKube specification sets the host group using the --set-host-group parameter. Instead, specify the new spec.oneagent.hostGroup field. If both settings are used, the new field takes precedence over the parameter.`

	versionRegex = `^\d+.\d+.\d+.\d{8}-\d{6}$`

	versionInvalidMessage = "The OneAgent's version is only valid in the format 'major.minor.patch.timestamp', e.g. 1.0.0.20240101-000000"
)

func conflictingOneAgentConfiguration(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	counter := 0
	if dk.ApplicationMonitoringMode() {
		counter += 1
	}

	if dk.CloudNativeFullstackMode() {
		counter += 1
	}

	if dk.ClassicFullStackMode() {
		counter += 1
	}

	if dk.HostMonitoringMode() {
		counter += 1
	}

	if counter > 1 {
		log.Info("requested dynakube has conflicting one agent configuration", "name", dk.Name, "namespace", dk.Namespace)

		return errorConflictingOneagentMode
	}

	return ""
}

func conflictingOneAgentNodeSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.NeedsOneAgent() || dk.FeatureEnableMultipleOsAgentsOnNode() {
		return ""
	}

	validDynakubes := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, validDynakubes, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	oneAgentNodeSelector := dk.OneAgentNodeSelector()
	conflictingDynakubes := make(map[string]bool)

	for _, item := range validDynakubes.Items {
		if item.Name == dk.Name {
			continue
		}

		if item.NeedsOneAgent() {
			if hasConflictingMatchLabels(oneAgentNodeSelector, item.OneAgentNodeSelector()) {
				log.Info("requested dynakube has conflicting OneAgent nodeSelector", "name", dk.Name, "namespace", dk.Namespace)

				conflictingDynakubes[item.Name] = true
			}
		}

		if item.LogMonitoring().IsEnabled() {
			if hasConflictingMatchLabels(oneAgentNodeSelector, item.LogMonitoring().NodeSelector) {
				log.Info("requested dynakube has conflicting LogMonitoring nodeSelector", "name", dk.Name, "namespace", dk.Namespace)

				conflictingDynakubes[item.Name] = true
			}
		}
	}

	if len(conflictingDynakubes) > 0 {
		return fmt.Sprintf(errorNodeSelectorConflict, mapKeysToString(conflictingDynakubes, ", "))
	}

	return ""
}

func mapKeysToString(m map[string]bool, sep string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return strings.Join(keys, sep)
}

func imageFieldSetWithoutCSIFlag(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.ApplicationMonitoringMode() {
		if !dk.NeedsCSIDriver() && len(dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage) > 0 {
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
	envVar := env.FindEnvVar(dk.GetOneAgentEnvironment(), oneagentEnableVolumeStorageEnvVarName)
	isSet = envVar != nil
	isEnabled = isSet && envVar.Value == "true"

	return
}

func unsupportedOneAgentImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if env.FindEnvVar(dk.GetOneAgentEnvironment(), oneagentInstallerScriptUrlEnvVarName) != nil ||
		env.FindEnvVar(dk.GetOneAgentEnvironment(), oneagentInstallerTokenEnvVarName) != nil {
		return warningOneAgentInstallerEnvVars
	}

	return ""
}

func conflictingOneAgentVolumeStorageSettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	volumeStorageEnabled, volumeStorageSet := hasOneAgentVolumeStorageEnabled(dk)
	if dk.NeedsReadOnlyOneAgents() && volumeStorageSet && !volumeStorageEnabled {
		return errorVolumeStorageReadOnlyModeConflict
	}

	return ""
}

func conflictingHostGroupSettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.HostGroupAsParam() != "" {
		return warningHostGroupConflict
	}

	return ""
}

func isOneAgentVersionValid(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	agentVersion := dk.CustomOneAgentVersion()
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
