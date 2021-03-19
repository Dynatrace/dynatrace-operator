package deploymentmetadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testOrchestratorID = "test-uid"

	testKey   = "test-key"
	testValue = "test-value"
)

func newTestDeploymentMetadata(_ *testing.T) *DeploymentMetadata {
	return NewDeploymentMetadata(testOrchestratorID)
}

func TestNewDeploymentMetadata(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)

	assert.Equal(t, testOrchestratorID, deploymentMetadata.OrchestratorID)
}

func TestDeploymentMetadata_asArgs(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsArgs()

	assert.Equal(t, []string{
		`--set-deployment-metadata=orchestration_tech=Operator`,
		`--set-deployment-metadata=script_version=snapshot`,
		`--set-deployment-metadata=orchestrator_id=` + testOrchestratorID,
	}, labels)
}

func TestDeploymentMetadata_asString(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsString()

	assert.Equal(t, `orchestration_tech=Operator;script_version=snapshot;orchestrator_id=`+testOrchestratorID, labels)
}

func TestDeploymentMetadata_asString_empty_agent(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsString()

	assert.Equal(t, `orchestration_tech=Operator;script_version=snapshot;orchestrator_id=`+testOrchestratorID, labels)
}

func TestFormatKeyValue(t *testing.T) {
	formattedArgument := formatKeyValue(testKey, testValue)
	assert.Equal(t, testKey+`=`+testValue, formattedArgument)
}

func TestFormatMetadataArgument(t *testing.T) {
	formattedArgument := formatMetadataArgument(testKey, testValue)
	assert.Equal(t, `--set-deployment-metadata=`+testKey+`=`+testValue, formattedArgument)
}
