package deploymentmetadata

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
)

type DeploymentMetadata struct {
	OrchestratorID string
	DeploymentType string
}

func GetOneAgentDeploymentType(dk dynakube.DynaKube) string {
	switch {
	case dk.OneAgent().HostMonitoringMode():
		return HostMonitoringDeploymentType
	case dk.OneAgent().CloudNativeFullstackMode():
		return CloudNativeDeploymentType
	case dk.OneAgent().ClassicFullStackMode():
		return ClassicFullStackDeploymentType
	case dk.OneAgent().ApplicationMonitoringMode():
		return ApplicationMonitoringDeploymentType
	}

	return ""
}

func NewDeploymentMetadata(orchestratorID string, dt string) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID, DeploymentType: dt}
}

func (metadata *DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(orchestrationTechKey, metadata.OrchestrationTech()),
		formatKeyValue(operatorScriptVersionKey, version.Version),
		formatKeyValue(orchestratorIDKey, metadata.OrchestratorID),
	}

	return strings.Join(res, ";")
}

func (metadata *DeploymentMetadata) OrchestrationTech() string {
	return fmt.Sprintf("%s-%s", orchestrationTech, metadata.DeploymentType)
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
