package deployment_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testOperatorScriptVersion = "0.9.5"
	testOrchestratorId        = "test-uid"
	testContainerImageVersion = "1.203.0.20200908-220956"

	testKey   = "test-key"
	testValue = "test-value"
)

func newTestDeploymentMetadata(_ *testing.T) *DeploymentMetadata {
	deploymentMetadata := NewDeploymentMetadata(
		testOperatorScriptVersion,
		testOrchestratorId,
		testContainerImageVersion)

	return deploymentMetadata
}

func TestNewDeploymentMetadata(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)

	assert.Equal(t, testOperatorScriptVersion, deploymentMetadata.OperatorScriptVersion)
	assert.Equal(t, testOrchestratorId, deploymentMetadata.OrchestratorID)
	assert.Equal(t, testContainerImageVersion, deploymentMetadata.ContainerImageVersion)
	assert.Equal(t, orchestrationTech, deploymentMetadata.OrchestrationTech)
}

func TestDeploymentMetadata_asArgs(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsArgs()

	assert.Equal(t, []string{
		`--set-deployment-metadata=orchestration_tech=Operator`,
		`--set-deployment-metadata=script_version=` + testOperatorScriptVersion,
		`--set-deployment-metadata=container_image_version=` + testContainerImageVersion,
		`--set-deployment-metadata=orchestrator_id=` + testOrchestratorId,
	}, labels)
}

func TestDeploymentMetadata_asString(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsString()

	assert.Equal(t,
		`orchestration_tech=Operator`+
			`;script_version=`+testOperatorScriptVersion+
			`;container_image_version=`+testContainerImageVersion+
			`;orchestrator_id=`+testOrchestratorId, labels)
}

func TestDeploymentMetadata_asString_empty_agent(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	deploymentMetadata.ContainerImageVersion = ""
	labels := deploymentMetadata.AsString()

	assert.Equal(t,
		`orchestration_tech=Operator`+
			`;script_version=`+testOperatorScriptVersion+
			`;orchestrator_id=`+testOrchestratorId, labels)
}

func TestFormatKeyValue(t *testing.T) {
	formattedArgument := formatKeyValue(testKey, testValue)
	assert.Equal(t, testKey+`=`+testValue, formattedArgument)
}

func TestFormatMetadataArgument(t *testing.T) {
	formattedArgument := formatMetadataArgument(testKey, testValue)
	assert.Equal(t, `--set-deployment-metadata=`+testKey+`=`+testValue, formattedArgument)
}
