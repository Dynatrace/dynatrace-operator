package deploymentmetadata

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/version"
)

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
)

type DeploymentMetadata struct {
	OrchestratorID string
}

func NewDeploymentMetadata(orchestratorID string) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID}
}

func (metadata *DeploymentMetadata) AsArgs() []string {
	return []string{
		formatMetadataArgument(keyOrchestrationTech, orchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}
}

func (metadata *DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(keyOrchestrationTech, orchestrationTech),
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
