package activegate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createActiveGateWithAnnotation(keyValues ...string) ActiveGate {
	activeGate := ActiveGate{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	for i := 0; i < len(keyValues); i += 2 {
		activeGate.Annotations[keyValues[i]] = keyValues[i+1]
	}

	return activeGate
}

func createActiveGateEmptyActiveGate() ActiveGate {
	activeGate := ActiveGate{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	return activeGate
}

func TestCreateActiveGateWithAnnotation(t *testing.T) {
	activeGate := createActiveGateWithAnnotation("test", "true")

	assert.Contains(t, activeGate.Annotations, "test")
	assert.Equal(t, "true", activeGate.Annotations["test"])

	activeGate = createActiveGateWithAnnotation("other test", "false")

	assert.Contains(t, activeGate.Annotations, "other test")
	assert.Equal(t, "false", activeGate.Annotations["other test"])
	assert.NotContains(t, activeGate.Annotations, "test")

	activeGate = createActiveGateWithAnnotation("test", "true", "other test", "false")

	assert.Contains(t, activeGate.Annotations, "other test")
	assert.Equal(t, "false", activeGate.Annotations["other test"])
	assert.Contains(t, activeGate.Annotations, "test")
	assert.Equal(t, "true", activeGate.Annotations["test"])
}

func testDeprecateDisableAnnotation(t *testing.T,
	newAnnotation string,
	deprecatedAnnotation string,
	propertyFunction func(activeGate ActiveGate) bool) {
	// New annotation works
	activeGate := createActiveGateWithAnnotation(newAnnotation, "false")

	assert.True(t, propertyFunction(activeGate))

	activeGate = createActiveGateWithAnnotation(newAnnotation, "true")

	assert.False(t, propertyFunction(activeGate))

	// Old annotation works
	activeGate = createActiveGateWithAnnotation(deprecatedAnnotation, "true")

	assert.True(t, propertyFunction(activeGate))

	activeGate = createActiveGateWithAnnotation(deprecatedAnnotation, "false")

	assert.False(t, propertyFunction(activeGate))

	// New annotation takes precedent
	activeGate = createActiveGateWithAnnotation(
		newAnnotation, "true",
		deprecatedAnnotation, "true")

	assert.False(t, propertyFunction(activeGate))

	activeGate = createActiveGateWithAnnotation(
		newAnnotation, "false",
		deprecatedAnnotation, "false")

	assert.True(t, propertyFunction(activeGate))

	// Default is false
	activeGate = createActiveGateWithAnnotation()

	assert.False(t, propertyFunction(activeGate))
}

func TestDeprecatedDisableAnnotations(t *testing.T) {
	t.Run(AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureActiveGateUpdates,
			AnnotationFeatureDisableActiveGateUpdates,
			func(activeGate ActiveGate) bool {
				return activeGate.FeatureDisableActiveGateUpdates()
			})
	})
}
func TestDefaultEnabledFeatureFlags(t *testing.T) {
	activeGate := createActiveGateEmptyActiveGate()

	assert.True(t, activeGate.FeatureAutomaticKubernetesApiMonitoring())
	assert.False(t, activeGate.FeatureDisableActiveGateUpdates())
}
