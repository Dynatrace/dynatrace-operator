package oneagent

import "fmt"

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyContainerImageVersion = "container_image_version"
	keyOrchestratorId        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
)

type deploymentMetadata struct {
	operatorScriptVersion string
	orchestratorId        string
	containerImageVersion string
	orchestrationTech     string
}

func newDeploymentMetadata(operatorScriptVersion string, orchestratorId string, containerImageVersion string) *deploymentMetadata {
	return &deploymentMetadata{
		operatorScriptVersion: operatorScriptVersion,
		orchestratorId:        orchestratorId,
		containerImageVersion: containerImageVersion,
		orchestrationTech:     orchestrationTech,
	}
}

func (metadata *deploymentMetadata) asArgs() []string {
	return []string{
		formatMetadataArgument(keyOrchestrationTech, metadata.orchestrationTech),
		formatMetadataArgument(keyOperatorScriptVersion, metadata.operatorScriptVersion),
		formatMetadataArgument(keyContainerImageVersion, metadata.containerImageVersion),
		formatMetadataArgument(keyOrchestratorId, metadata.orchestratorId),
	}
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
