package deploymentmetadata

import (
	"fmt"
	"strings"
)

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyContainerImageVersion = "container_image_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
)

type DeploymentMetadata struct {
	OperatorScriptVersion string
	OrchestratorID        string
	ContainerImageVersion string
	OrchestrationTech     string
}

func NewDeploymentMetadata(operatorScriptVersion string, orchestratorID string, containerImageVersion string) *DeploymentMetadata {
	return &DeploymentMetadata{
		OperatorScriptVersion: operatorScriptVersion,
		OrchestratorID:        orchestratorID,
		ContainerImageVersion: containerImageVersion,
		OrchestrationTech:     orchestrationTech,
	}
}

func (metadata *DeploymentMetadata) AsArgs() []string {
	return []string{
		formatMetadataArgument(keyOrchestrationTech, metadata.OrchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, metadata.OperatorScriptVersion),
		formatMetadataArgument(keyContainerImageVersion, metadata.ContainerImageVersion),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}
}

func (metadata *DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(keyOrchestrationTech, metadata.OrchestrationTech),
		formatKeyValue(keyOperatorScriptVersion, metadata.OperatorScriptVersion),
		formatKeyValue(keyContainerImageVersion, metadata.ContainerImageVersion),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
	}

	if metadata.ContainerImageVersion == "" {
		res = append(res[:2], res[3:]...)
	}

	return strings.Join(res, ";")
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
