package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingOneagentMode = `The DynaKube specification attempts to use multiple OneAgent modes simultaneously, which is not supported.`

	errorImageFieldSetWithoutCSIFlag = `The DynaKube specification attempts to enable ApplicationMonitoring/CloudNativeFullstack mode and retrieve the respective codeModules image, but the CSI driver and/or node image pull is not enabled.`

	errorImagePullRequiresCodeModulesImage = `The DynaKube specification enables node image pull, but the code modules image is not set.`

	errorNodeSelectorConflict = `The Dynakube specification conflicts with another Dynakube's OneAgent or Standalone-LogMonitoring. Only one Agent per node is supported.
Use a nodeSelector to avoid this conflict. Conflicting DynaKubes: %s`

	errorVolumeStorageReadOnlyModeConflict = `The DynaKube specification specifies a read-only host file system while OneAgent has volume storage enabled.`

	warningOneAgentInstallerEnvVars = `The environment variables ONEAGENT_INSTALLER_SCRIPT_URL and ONEAGENT_INSTALLER_TOKEN are only relevant for an unsupported image type. Please ensure you are using a supported image.`

	warningHostGroupConflict = `The DynaKube specification sets the host group using the --set-host-group parameter. Instead, specify the new spec.oneagent.hostGroup field. If both settings are used, the new field takes precedence over the parameter.`

	warningDeprecatedAutoUpdate = `AutoUpdate field is deprecated. The feature is still available by configuring the DynaTrace tenant. Please visit our documentation for more details.`

	versionRegex = `^\d+.\d+.\d+.\d{8}-\d{6}$`

	versionInvalidMessage = "The OneAgent's version is only valid in the format 'major.minor.patch.timestamp', e.g. 1.0.0.20240101-000000"

	errorDuplicateOneAgentArgument = "%s has been provided multiple times. Only --set-host-property and --set-host-tag arguments may be provided multiple times."

	errorHostIDSourceArgumentInCloudNative = "Setting --set-host-id-source in CloudNativFullstack mode is not allowed."

	errorSameHostTagMultipleTimes = "Providing the same tag(s) (%s) multiple times with --set-host-tag is not allowed."
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
	return dk1.LogMonitoring().IsStandalone() && dk1.APIURL() == dk2.APIURL() &&
		(dk2.OneAgent().IsDaemonsetRequired() || dk2.LogMonitoring().IsStandalone()) &&
		hasConflictingMatchLabels(dk1.OneAgent().GetNodeSelector(dk1.LogMonitoring().GetNodeSelector()),
			dk2.OneAgent().GetNodeSelector(dk2.LogMonitoring().GetNodeSelector()))
}

func hasOneAgentSelectorConflict(dk1, dk2 *dynakube.DynaKube) bool {
	return dk1.OneAgent().IsDaemonsetRequired() &&
		(dk2.OneAgent().IsDaemonsetRequired() || dk2.LogMonitoring().IsStandalone() && dk1.APIURL() == dk2.APIURL()) &&
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

func imageFieldSetWithoutCSIFlag(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if !v.modules.CSIDriver && !dk.FF().IsNodeImagePull() {
		if dk.OneAgent().IsApplicationMonitoringMode() && len(dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage) > 0 {
			return errorImageFieldSetWithoutCSIFlag
		}

		if dk.OneAgent().IsCloudNativeFullstackMode() && len(dk.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage) > 0 {
			return errorImageFieldSetWithoutCSIFlag
		}
	}

	return ""
}

func missingCodeModulesImage(_ context.Context, v *Validator, dk *dynakube.DynaKube) string {
	if dk.OneAgent().IsAppInjectionNeeded() &&
		dk.FF().IsNodeImagePull() &&
		len(dk.OneAgent().GetCustomCodeModulesImage()) == 0 {
		return errorImagePullRequiresCodeModulesImage
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
	if env.FindEnvVar(dk.OneAgent().GetEnvironment(), oneagentInstallerScriptURLEnvVarName) != nil ||
		env.FindEnvVar(dk.OneAgent().GetEnvironment(), oneagentInstallerTokenEnvVarName) != nil {
		return warningOneAgentInstallerEnvVars
	}

	return ""
}

func conflictingOneAgentVolumeStorageSettings(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	volumeStorageEnabled, volumeStorageSet := hasOneAgentVolumeStorageEnabled(dk)
	if dk.OneAgent().IsReadOnlyFSSupported() && volumeStorageSet && !volumeStorageEnabled {
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

func deprecatedAutoUpdate(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()
	if oa.IsClassicFullStackMode() && dk.RemovedFields().AutoUpdate.Get() != nil {
		return warningDeprecatedAutoUpdate
	}

	if oa.IsHostMonitoringMode() && dk.RemovedFields().AutoUpdate.Get() != nil {
		return warningDeprecatedAutoUpdate
	}

	if oa.IsCloudNativeFullstackMode() && dk.RemovedFields().AutoUpdate.Get() != nil {
		return warningDeprecatedAutoUpdate
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

func duplicateOneAgentArguments(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	args := dk.OneAgent().GetArgumentsMap()
	if args == nil {
		return ""
	}

	for key, values := range args {
		if key != "--set-host-property" && key != "--set-host-tag" && len(values) > 1 {
			return fmt.Sprintf(errorDuplicateOneAgentArgument, key)
		} else if key == "--set-host-tag" {
			if duplicatedTags := findDuplicates(values); len(duplicatedTags) > 0 {
				return fmt.Sprintf(errorSameHostTagMultipleTimes, duplicatedTags)
			}
		}
	}

	return ""
}

func forbiddenHostIDSourceArgument(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	args := dk.OneAgent().GetArgumentsMap()
	if args == nil {
		return ""
	}

	for key := range args {
		if dk.OneAgent().IsCloudNativeFullstackMode() && key == "--set-host-id-source" {
			return errorHostIDSourceArgumentInCloudNative
		}
	}

	return ""
}

func findDuplicates[S ~[]E, E comparable](s S) []E {
	seen := make(map[E]int)

	var duplicates []E

	const seenTwice = 2

	for _, val := range s {
		seen[val]++
		if seen[val] == seenTwice {
			duplicates = append(duplicates, val)
		}
	}

	return duplicates
}
