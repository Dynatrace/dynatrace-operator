package oneagent

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEnvVars(t *testing.T) {
	log := logger.NewDTLogger()
	reservedVariable := "DT_K8S_NODE_NAME"
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
				Env: []corev1.EnvVar{
					{
						Name:  testName,
						Value: testValue,
					},
					{
						Name:  reservedVariable,
						Value: testValue,
					},
				},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.NotEmpty(t, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, testName, testValue, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, reservedVariable, testValue, podSpecs.Containers[0].Env)
}

func assertHasEnvVar(t *testing.T, expectedName string, expectedValue string, envVars []corev1.EnvVar) {
	hasVariable := false
	for _, env := range envVars {
		if env.Name == expectedName {
			hasVariable = true
			assert.Equal(t, expectedValue, env.Value)
		}
	}
	assert.True(t, hasVariable)
}
