package oneagent

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

func newTestDeploymentMetadata(_ *testing.T) *deploymentMetadata {
	deploymentMetadata := newDeploymentMetadata(
		testOperatorScriptVersion,
		testOrchestratorId,
		testContainerImageVersion)

	return deploymentMetadata
}

func TestNewDeploymentMetadata(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)

	assert.Equal(t, testOperatorScriptVersion, deploymentMetadata.operatorScriptVersion)
	assert.Equal(t, testOrchestratorId, deploymentMetadata.orchestratorId)
	assert.Equal(t, testContainerImageVersion, deploymentMetadata.containerImageVersion)
	assert.Equal(t, orchestrationTech, deploymentMetadata.orchestrationTech)
}

func TestDeploymentMetadata_asArgs(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.asArgs()

	assert.Equal(t, []string{
		`--set-deployment-metadata=orchestration_tech=Operator`,
		`--set-deployment-metadata=script_version=` + testOperatorScriptVersion,
		`--set-deployment-metadata=container_image_version=` + testContainerImageVersion,
		`--set-deployment-metadata=orchestrator_id=` + testOrchestratorId,
	}, labels)
}

func TestFormatMetadataArgument(t *testing.T) {
	formattedArgument := formatMetadataArgument(testKey, testValue)
	assert.Equal(t, `--set-deployment-metadata=`+testKey+`=`+testValue, formattedArgument)
}
