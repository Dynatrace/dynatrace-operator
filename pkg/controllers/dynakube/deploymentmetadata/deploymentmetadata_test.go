package deploymentmetadata

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
)

const (
	testOrchestratorID = "test-uid"

	testKey      = "test-key"
	testValue    = "test-value"
	testMetaData = "testMetaData"
)

func newTestDeploymentMetadata(_ *testing.T) *DeploymentMetadata {
	return NewDeploymentMetadata(testOrchestratorID, testMetaData)
}

func TestNewDeploymentMetadata(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)

	assert.Equal(t, testOrchestratorID, deploymentMetadata.OrchestratorID)
}

func TestDeploymentMetadata_asString(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsString()

	assert.Equal(t, `orchestration_tech=Operator-`+testMetaData+`;script_version=snapshot;orchestrator_id=`+testOrchestratorID, labels)
}

func TestDeploymentMetadata_asString_empty_agent(t *testing.T) {
	deploymentMetadata := newTestDeploymentMetadata(t)
	labels := deploymentMetadata.AsString()

	assert.Equal(t, `orchestration_tech=Operator-`+testMetaData+`;script_version=snapshot;orchestrator_id=`+testOrchestratorID, labels)
}

func TestFormatKeyValue(t *testing.T) {
	formattedArgument := formatKeyValue(testKey, testValue)
	assert.Equal(t, testKey+`=`+testValue, formattedArgument)
}

func TestGetOneAgentDeploymentType(t *testing.T) {
	tests := []struct {
		oneAgentSpec           dynakube.OneAgentSpec
		expectedDeploymentType string
	}{
		{dynakube.OneAgentSpec{HostMonitoring: &dynakube.HostInjectSpec{}}, HostMonitoringDeploymentType},
		{dynakube.OneAgentSpec{ClassicFullStack: &dynakube.HostInjectSpec{}}, ClassicFullStackDeploymentType},
		{dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}}, CloudNativeDeploymentType},
		{dynakube.OneAgentSpec{ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{}}, ApplicationMonitoringDeploymentType},
	}

	for _, test := range tests {
		dynakube := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: test.oneAgentSpec,
			},
		}
		deploymentType := GetOneAgentDeploymentType(*dynakube)
		assert.Equal(t, test.expectedDeploymentType, deploymentType)
	}
}
