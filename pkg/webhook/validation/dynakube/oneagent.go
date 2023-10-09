package dynakube

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	errorConflictingOneagentMode = `The DynaKube's specification tries to use multiple oneagent modes at the same time, which is not supported.
`
	errorImageFieldSetWithoutCSIFlag = `The DynaKube's specification tries to enable ApplicationMonitoring mode and get the respective image, but the CSI driver is not enabled.`

	errorNodeSelectorConflict = `The DynaKube's specification tries to specify a nodeSelector conflicts with an another Dynakube's nodeSelector, which is not supported.
The conflicting Dynakube: %s
`
	errorVolumeStorageReadOnlyModeConflict = `The DynaKube's specification specifies a read-only host file system and OneAgent has volume storage enabled.`

	warningIneffectiveFeatureFlag = `Feature flag %s has no effect in classic full stack mode.`
)

func conflictingOneAgentConfiguration(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	counter := 0
	if dynakube.ApplicationMonitoringMode() {
		counter += 1
	}
	if dynakube.CloudNativeFullstackMode() {
		counter += 1
	}
	if dynakube.ClassicFullStackMode() {
		counter += 1
	}
	if dynakube.HostMonitoringMode() {
		counter += 1
	}
	if counter > 1 {
		log.Info("requested dynakube has conflicting one agent configuration", "name", dynakube.Name, "namespace", dynakube.Namespace)
		return errorConflictingOneagentMode
	}
	return ""
}

func conflictingReadOnlyFilesystemAndMultipleOsAgentsOnNode(_ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.FeatureDisableReadOnlyOneAgent() && dynakube.FeatureEnableMultipleOsAgentsOnNode() {
		return "Multiple OsAgents require readonly host filesystem"
	}
	return ""
}

func conflictingNodeSelector(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedsOneAgent() || dynakube.FeatureEnableMultipleOsAgentsOnNode() {
		return ""
	}
	validDynakubes := &dynatracev1beta1.DynaKubeList{}
	if err := dv.clt.List(context.TODO(), validDynakubes); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())
		return ""
	}
	for _, item := range validDynakubes.Items {
		if !item.NeedsOneAgent() {
			continue
		}
		nodeSelectorMap := dynakube.NodeSelector()
		validNodeSelectorMap := item.NodeSelector()
		if item.Name != dynakube.Name {
			if hasConflictingMatchLabels(nodeSelectorMap, validNodeSelectorMap) {
				log.Info("requested dynakube has conflicting nodeSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
				return fmt.Sprintf(errorNodeSelectorConflict, item.Name)
			}
		}
	}
	return ""
}

func imageFieldSetWithoutCSIFlag(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ApplicationMonitoringMode() {
		if !dynakube.NeedsCSIDriver() && len(dynakube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage) > 0 {
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

func hasOneAgentVolumeStorageEnabled(dynakube *dynatracev1beta1.DynaKube) (isEnabled bool, isSet bool) {
	envVar := kubeobjects.FindEnvVar(dynakube.GetOneAgentEnvironment(), oneagentEnableVolumeStorageEnvVarName)
	isSet = envVar != nil
	isEnabled = isSet && envVar.Value == "true"
	return
}

func conflictingOneAgentVolumeStorageSettings(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	volumeStorageEnabled, volumeStorageSet := hasOneAgentVolumeStorageEnabled(dynakube)
	if dynakube.NeedsReadOnlyOneAgents() && volumeStorageSet && !volumeStorageEnabled {
		return errorVolumeStorageReadOnlyModeConflict
	}
	return ""
}

func ineffectiveReadOnlyHostFsFeatureFlag(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ClassicFullStackMode() {
		if _, hasOneAgentReadOnlyFeatureFlag := dynakube.Annotations[dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent]; hasOneAgentReadOnlyFeatureFlag {
			return readonlyHostFsFlagWarning(dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent)
		}
		if _, hasOneAgentReadOnlyFeatureFlag := dynakube.Annotations[dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent]; hasOneAgentReadOnlyFeatureFlag {
			return readonlyHostFsFlagWarning(dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent)
		}
	}
	return ""
}

func readonlyHostFsFlagWarning(featureFlag string) string {
	return fmt.Sprintf(warningIneffectiveFeatureFlag, featureFlag)
}
