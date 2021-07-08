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
}

func NewDeploymentMetadata(orchestratorID string, instance dynatracev1alpha1.DynaKube) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID,
		DynaKube: &instance}
}

func (metadata *DeploymentMetadata) AsArgs() []string {
	codeModulesDeployed := "false"
	if metadata.DynaKube.Spec.CodeModules.Enabled {
		codeModulesDeployed = "true"
	}

	return []string{
		formatMetadataArgument(keyOrchestrationTech, orchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
		formatMetadataArgument(keyOSAgentDeployed, ""),
		formatMetadataArgument(keyCodeModulesDeployed, codeModulesDeployed),
		formatMetadataArgument(keyKubemonDeployed, ""),
		formatMetadataArgument(keyRoutingDeployed, ""),
	}
}

func (metadata *DeploymentMetadata) AsString() string {
	codeModulesDeployed := "false"
	if metadata.DynaKube.Spec.CodeModules.Enabled {
		codeModulesDeployed = "true"
	}

	res := []string{
		formatKeyValue(keyOrchestrationTech, orchestrationTech),
		formatKeyValue(keyOperatorScriptVersion, version.Version),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
		formatMetadataArgument(keyOSAgentDeployed, ""),
		formatMetadataArgument(keyCodeModulesDeployed, codeModulesDeployed),
		formatMetadataArgument(keyKubemonDeployed, ""),
		formatMetadataArgument(keyRoutingDeployed, ""),
	}

	return strings.Join(res, ";")
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
