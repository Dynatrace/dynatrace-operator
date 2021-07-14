package deploymentmetadata

import (
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/version"
)

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
	keyOSAgentDeployed       = "os_agent_deployed"
	keyCodeModulesDeployed   = "code_modules_deployed"
	keyKubemonDeployed       = "kubemon_deployed"
	keyRoutingDeployed       = "routing_deployed"
)

type DeploymentMetadata struct {
	OrchestratorID string
	*dynatracev1alpha1.DynaKube
	codeModulesDeployed string
	kubemonDeployed     string
	routingDeployed     string
	osAgentDeployed     string
}

func NewDeploymentMetadata(orchestratorID string, instance dynatracev1alpha1.DynaKube) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID,
		DynaKube: &instance,
	}
}

func (metadata *DeploymentMetadata) AsArgs() []string {
	checkDeployments(metadata)

	res := []string{
		formatMetadataArgument(keyOrchestrationTech, orchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}

	if metadata.osAgentDeployed != "" {
		res = append(res, formatMetadataArgument(keyOSAgentDeployed, metadata.osAgentDeployed))
	}

	if metadata.codeModulesDeployed != "" {
		res = append(res, formatMetadataArgument(keyCodeModulesDeployed, metadata.codeModulesDeployed))
	}

	if metadata.kubemonDeployed != "" {
		res = append(res, formatMetadataArgument(keyKubemonDeployed, metadata.kubemonDeployed))
	}

	if metadata.routingDeployed != "" {
		res = append(res, formatMetadataArgument(keyRoutingDeployed, metadata.routingDeployed))
	}

	return res
}

func (metadata *DeploymentMetadata) AsString() string {
	checkDeployments(metadata)

	res := []string{
		formatKeyValue(keyOrchestrationTech, orchestrationTech),
		formatKeyValue(keyOperatorScriptVersion, version.Version),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
	}

	if metadata.osAgentDeployed != "" {
		res = append(res, formatKeyValue(keyOSAgentDeployed, metadata.osAgentDeployed))
	}

	if metadata.codeModulesDeployed != "" {
		res = append(res, formatKeyValue(keyCodeModulesDeployed, metadata.codeModulesDeployed))
	}

	if metadata.kubemonDeployed != "" {
		res = append(res, formatKeyValue(keyKubemonDeployed, metadata.kubemonDeployed))
	}

	if metadata.routingDeployed != "" {
		res = append(res, formatKeyValue(keyRoutingDeployed, metadata.routingDeployed))
	}

	return strings.Join(res, ";")
}

func checkDeployments(metadata *DeploymentMetadata) {
	if metadata.DynaKube.Spec.CodeModules.Enabled {
		metadata.codeModulesDeployed = "true"
	}

	if metadata.DynaKube.Spec.KubernetesMonitoringSpec.Enabled {
		metadata.kubemonDeployed = "true"
	}

	if metadata.DynaKube.Spec.RoutingSpec.Enabled {
		metadata.routingDeployed = "true"
	}

	if metadata.DynaKube.Spec.InfraMonitoring.Enabled {
		metadata.osAgentDeployed = "true"
	}

	if metadata.DynaKube.Spec.ClassicFullStack.Enabled {
		metadata.osAgentDeployed = "true"
		metadata.codeModulesDeployed = ""
	}
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
