package deploymentmetadata

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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
		oneAgentSpec           dynatracev1beta1.OneAgentSpec
		expectedDeploymentType string
	}{
		{dynatracev1beta1.OneAgentSpec{HostMonitoring: &dynatracev1beta1.HostInjectSpec{}}, HostMonitoringDeploymentType},
		{dynatracev1beta1.OneAgentSpec{ClassicFullStack: &dynatracev1beta1.HostInjectSpec{}}, ClassicFullStackDeploymentType},
		{dynatracev1beta1.OneAgentSpec{CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{}}, CloudNativeDeploymentType},
		{dynatracev1beta1.OneAgentSpec{ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{}}, ApplicationMonitoringDeploymentType},
	}

	for _, test := range tests {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: test.oneAgentSpec,
			},
		}
		deploymentType := GetOneAgentDeploymentType(*dynakube)
		assert.Equal(t, test.expectedDeploymentType, deploymentType)
	}
}
