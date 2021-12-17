package deploymentmetadata

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"

	DeploymentTypeApplicationMonitoring = "application_monitoring"
	DeploymentTypeFullStack             = "classic_fullstack"
	DeploymentTypeCloudNative           = "cloud_native_fullstack"
	DeploymentTypeHostMonitoring        = "host_monitoring"
	DeploymentTypeActiveGate            = "active_gate"
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
		formatMetadataArgument(keyOrchestrationTech, metadata.OrchestrationTech()),
		formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}
}

func (metadata *DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(keyOrchestrationTech, metadata.OrchestrationTech()),
		formatKeyValue(keyOperatorScriptVersion, version.Version),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
	}

	return strings.Join(res, ";")
}

func (metadata *DeploymentMetadata) OrchestrationTech() string {
	return fmt.Sprintf("%s-%s", orchestrationTech, metadata.DeploymentType)
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}
