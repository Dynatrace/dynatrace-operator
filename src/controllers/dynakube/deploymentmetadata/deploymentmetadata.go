package deploymentmetadata

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/version"
)

type DeploymentMetadata struct {
	OrchestratorID string
	DeploymentType string
}

func GetOneAgentDeploymentType(dynakube dynatracev1beta1.DynaKube) string {
	switch {
	case dynakube.HostMonitoringMode():
		return DeploymentTypeHostMonitoring
	case dynakube.CloudNativeFullstackMode():
		return DeploymentTypeCloudNative
	case dynakube.ClassicFullStackMode():
		return DeploymentTypeFullStack
	case dynakube.ApplicationMonitoringMode():
		return DeploymentTypeApplicationMonitoring
	}

	return ""
}

func NewDeploymentMetadata(orchestratorID string, dt string) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID, DeploymentType: dt}
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
