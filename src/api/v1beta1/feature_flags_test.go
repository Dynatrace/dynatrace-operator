package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDynakubeWithAnnotation(keyValues ...string) DynaKube {
	dynakube := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	for i := 0; i < len(keyValues); i += 2 {
		dynakube.Annotations[keyValues[i]] = keyValues[i+1]
	}

	return dynakube
}

func TestCreateDynakubeWithAnnotation(t *testing.T) {
	dynakube := createDynakubeWithAnnotation("test", "true")

	assert.Contains(t, dynakube.Annotations, "test")
	assert.Equal(t, dynakube.Annotations["test"], "true")

	dynakube = createDynakubeWithAnnotation("other test", "false")

	assert.Contains(t, dynakube.Annotations, "other test")
	assert.Equal(t, dynakube.Annotations["other test"], "false")
	assert.NotContains(t, dynakube.Annotations, "test")

	dynakube = createDynakubeWithAnnotation("test", "true", "other test", "false")

	assert.Contains(t, dynakube.Annotations, "other test")
	assert.Equal(t, dynakube.Annotations["other test"], "false")
	assert.Contains(t, dynakube.Annotations, "test")
	assert.Equal(t, dynakube.Annotations["test"], "true")
}

func testDeprecateDisableAnnotation(t *testing.T,
	newAnnotation string,
	deprecatedAnnotation string,
	propertyFunction func(dynakube DynaKube) bool) {
	// New annotation works
	dynakube := createDynakubeWithAnnotation(newAnnotation, "false")

	assert.True(t, propertyFunction(dynakube))

	dynakube = createDynakubeWithAnnotation(newAnnotation, "true")

	assert.False(t, propertyFunction(dynakube))

	// Old annotation works
	dynakube = createDynakubeWithAnnotation(deprecatedAnnotation, "true")

	assert.True(t, propertyFunction(dynakube))

	dynakube = createDynakubeWithAnnotation(deprecatedAnnotation, "false")

	assert.False(t, propertyFunction(dynakube))

	// New annotation takes precedent
	dynakube = createDynakubeWithAnnotation(
		newAnnotation, "true",
		deprecatedAnnotation, "true")

	assert.False(t, propertyFunction(dynakube))

	dynakube = createDynakubeWithAnnotation(
		newAnnotation, "false",
		deprecatedAnnotation, "false")

	assert.True(t, propertyFunction(dynakube))

	// Default is false
	dynakube = createDynakubeWithAnnotation()

	assert.False(t, propertyFunction(dynakube))
}

func TestDeprecatedDisableAnnotations(t *testing.T) {
	t.Run(AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureActiveGateUpdates,
			AnnotationFeatureDisableActiveGateUpdates,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableActiveGateUpdates()
			})
	})
	t.Run(AnnotationFeatureHostsRequests, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureHostsRequests,
			AnnotationFeatureDisableHostsRequests,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableHostsRequests()
			})
	})
	t.Run(AnnotationFeatureWebhookReinvocationPolicy, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureWebhookReinvocationPolicy,
			AnnotationFeatureDisableWebhookReinvocationPolicy,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableWebhookReinvocationPolicy()
			})
	})
	t.Run(AnnotationFeatureMetadataEnrichment, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureMetadataEnrichment,
			AnnotationFeatureDisableMetadataEnrichment,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableMetadataEnrichment()
			})
	})
	t.Run(AnnotationFeatureReadOnlyOneAgent, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureReadOnlyOneAgent,
			AnnotationFeatureDisableReadOnlyOneAgent,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableReadOnlyOneAgent()
			})
	})
	t.Run(AnnotationFeatureReadOnlyOneAgent, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureActiveGateRawImage,
			AnnotationFeatureDisableActiveGateRawImage,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableActivegateRawImage()
			})
	})
}

func TestDeprecatedEnableAnnotations(t *testing.T) {
	// New annotation works
	dynakube := createDynakubeWithAnnotation(AnnotationFeatureActiveGateAuthToken, "false")

	assert.False(t, dynakube.FeatureActiveGateAuthToken())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureActiveGateAuthToken, "true")

	assert.True(t, dynakube.FeatureActiveGateAuthToken())

	// Old annotation works
	dynakube = createDynakubeWithAnnotation(AnnotationFeatureEnableActiveGateAuthToken, "false")

	assert.False(t, dynakube.FeatureActiveGateAuthToken())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureEnableActiveGateAuthToken, "true")

	assert.True(t, dynakube.FeatureActiveGateAuthToken())

	// New annotation takes precedent
	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureActiveGateAuthToken, "true",
		AnnotationFeatureEnableActiveGateAuthToken, "false")

	assert.True(t, dynakube.FeatureActiveGateAuthToken())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureActiveGateAuthToken, "false",
		AnnotationFeatureEnableActiveGateAuthToken, "true")

	assert.False(t, dynakube.FeatureActiveGateAuthToken())

	// Default is false
	dynakube = createDynakubeWithAnnotation()
	assert.False(t, dynakube.FeatureActiveGateAuthToken())
}

func TestMaxMountAttempts(t *testing.T) {
	dynakube := createDynakubeWithAnnotation(
		AnnotationFeatureMaxMountAttempts, "5")

	assert.Equal(t, 5, *dynakube.FeatureMaxCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureMaxMountAttempts, "3")

	assert.Equal(t, 3, *dynakube.FeatureMaxCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation()

	assert.Nil(t, dynakube.FeatureMaxCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureMaxMountAttempts, "a")

	assert.Nil(t, dynakube.FeatureMaxCsiMountAttempts())
}
