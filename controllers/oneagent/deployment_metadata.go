package oneagent

import "fmt"

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyContainerImageVersion = "container_image_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
)

type deploymentMetadata struct {
	operatorScriptVersion string
	orchestratorID        string
	containerImageVersion string
	orchestrationTech     string
}

func newDeploymentMetadata(operatorScriptVersion string, orchestratorId string, containerImageVersion string) *deploymentMetadata {
	return &deploymentMetadata{
		operatorScriptVersion: operatorScriptVersion,
		orchestratorID:        orchestratorId,
		containerImageVersion: containerImageVersion,
		orchestrationTech:     orchestrationTech,
	}
}

func (metadata *deploymentMetadata) asArgs() []string {
	return []string{
		formatMetadataArgument(keyOrchestrationTech, metadata.orchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, metadata.operatorScriptVersion),
		formatMetadataArgument(keyContainerImageVersion, metadata.containerImageVersion),
		formatMetadataArgument(keyOrchestratorID, metadata.orchestratorID),
	}
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
