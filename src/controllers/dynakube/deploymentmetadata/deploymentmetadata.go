package deploymentmetadata

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/version"
)

type DeploymentMetadata struct {
	OrchestratorID string
	DeploymentType string
}

func GetOneAgentDeploymentType(dynakube dynatracev1beta1.DynaKube) string {
	switch {
	case dynakube.HostMonitoringMode():
		return HostMonitoringDeploymentType
	case dynakube.CloudNativeFullstackMode():
		return CloudNativeDeploymentType
	case dynakube.ClassicFullStackMode():
		return ClassicFullStackDeploymentType
	case dynakube.ApplicationMonitoringMode():
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
