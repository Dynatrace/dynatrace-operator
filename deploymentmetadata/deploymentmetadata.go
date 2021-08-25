package deploymentmetadata

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/version"
)

const (
	orchestrationTech = "operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"

	DeploymentTypeCodeModules = "code_modules"
	DeploymentTypeFS          = "classic_full_stack"
	DeploymentTypeIS          = "infrastructure"
	DeploymentTypeAG          = "active_gate"
)

type DeploymentMetadata struct {
	OrchestratorID string
	DeploymentType string
}

func NewDeploymentMetadata(orchestratorID string, dt string) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID, DeploymentType: dt}
}

func (metadata *DeploymentMetadata) AsArgs() []string {
	return []string{
		formatMetadataArgument(keyOrchestrationTech, fmt.Sprintf("%s-%s", orchestrationTech, metadata.DeploymentType)),
		formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}
}

func (metadata *DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(keyOrchestrationTech, fmt.Sprintf("%s-%s", orchestrationTech, metadata.DeploymentType)),
		formatKeyValue(keyOperatorScriptVersion, version.Version),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
	}

	return strings.Join(res, ";")
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
