package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube" //nolint:staticcheck
	dynakubev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingOneagentMode = `The DynaKube's specification tries to use multiple oneagent modes at the same time, which is not supported.
`
	errorImageFieldSetWithoutCSIFlag = `The DynaKube's specification tries to enable ApplicationMonitoring mode and get the respective image, but the CSI driver is not enabled.`

	errorNodeSelectorConflict = `The DynaKube specification attempts to deploy a %s, which conflicts with another DynaKube's %s. Only one Agent per node is supported.
Use a nodeSelector to avoid this conflict. Conflicting DynaKube: %s`

	errorVolumeStorageReadOnlyModeConflict = `The DynaKube's specification specifies a read-only host file system and OneAgent has volume storage enabled.`

	warningOneAgentInstallerEnvVars = `Environment variables ONEAGENT_INSTALLER_SCRIPT_URL and ONEAGENT_INSTALLER_TOKEN are only relevant for an unsupported image type. Please make sure you are using a supported image.`

	warningHostGroupConflict = `DynaKube's specification sets the host group using --set-host-group parameter. Instead, specify the new spec.oneagent.hostGroup field. If you use both settings, the new field precedes the parameter.`

	oneAgentComponentName = "OneAgent"

	logModuleComponentName = "LogModule"
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

func conflictingNodeSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.NeedsOneAgent() || dk.FeatureEnableMultipleOsAgentsOnNode() {
		return ""
	}

	validDynakubes := &dynakubev1beta3.DynaKubeList{}
	if err := dv.apiReader.List(ctx, validDynakubes, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	oneAgentNodeSelector := dk.NodeSelector()

	for _, item := range validDynakubes.Items {
		if item.Name == dk.Name {
			continue
		}

		if item.NeedsOneAgent() {
			if hasConflictingMatchLabels(oneAgentNodeSelector, item.OneAgentNodeSelector()) {
				log.Info("requested dynakube has conflicting OneAgent nodeSelector", "name", dk.Name, "namespace", dk.Namespace)

				return fmt.Sprintf(errorNodeSelectorConflict, oneAgentComponentName, oneAgentComponentName, item.Name)
			}
		}

		if item.NeedsLogModule() {
			if hasConflictingMatchLabels(oneAgentNodeSelector, item.LogModuleNodeSelector()) {
				log.Info("requested dynakube has conflicting LogModule nodeSelector", "name", dk.Name, "namespace", dk.Namespace)

				return fmt.Sprintf(errorNodeSelectorConflict, oneAgentComponentName, logModuleComponentName, item.Name)
			}
		}
	}

	return ""
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

func validateOneAgentVersionIsSemVerCompliant(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	agentVersion := dk.CustomOneAgentVersion()
	if agentVersion == "" {
		return ""
	}

	version := "v" + agentVersion
	if !(semver.IsValid(version) && semver.Prerelease(version) == "" && semver.Build(version) == "" && len(strings.Split(version, ".")) == 3) {
		return "Only semantic versions in the form of major.minor.patch (e.g. 1.0.0) are allowed!"
	}

	return ""
}
