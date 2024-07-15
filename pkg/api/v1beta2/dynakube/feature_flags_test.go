package dynakube

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDynakubeWithAnnotation(keyValues ...string) DynaKube {
	dk := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	for i := 0; i < len(keyValues); i += 2 {
		dk.Annotations[keyValues[i]] = keyValues[i+1]
	}

	return dk
}

func createDynakubeEmptyDynakube() DynaKube {
	dk := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	return dk
}

func TestCreateDynakubeWithAnnotation(t *testing.T) {
	dk := createDynakubeWithAnnotation("test", "true")

	assert.Contains(t, dk.Annotations, "test")
	assert.Equal(t, "true", dk.Annotations["test"])

	dk = createDynakubeWithAnnotation("other test", "false")

	assert.Contains(t, dk.Annotations, "other test")
	assert.Equal(t, "false", dk.Annotations["other test"])
	assert.NotContains(t, dk.Annotations, "test")

	dk = createDynakubeWithAnnotation("test", "true", "other test", "false")

	assert.Contains(t, dk.Annotations, "other test")
	assert.Equal(t, "false", dk.Annotations["other test"])
	assert.Contains(t, dk.Annotations, "test")
	assert.Equal(t, "true", dk.Annotations["test"])
}

func testDeprecateDisableAnnotation(t *testing.T,
	newAnnotation string,
	deprecatedAnnotation string,
	propertyFunction func(dk DynaKube) bool) {
	// New annotation works
	dk := createDynakubeWithAnnotation(newAnnotation, "false")

	assert.True(t, propertyFunction(dk))

	dk = createDynakubeWithAnnotation(newAnnotation, "true")

	assert.False(t, propertyFunction(dk))

	// Old annotation works
	dk = createDynakubeWithAnnotation(deprecatedAnnotation, "true")

	assert.True(t, propertyFunction(dk))

	dk = createDynakubeWithAnnotation(deprecatedAnnotation, "false")

	assert.False(t, propertyFunction(dk))

	// New annotation takes precedent
	dk = createDynakubeWithAnnotation(
		newAnnotation, "true",
		deprecatedAnnotation, "true")

	assert.False(t, propertyFunction(dk))

	dk = createDynakubeWithAnnotation(
		newAnnotation, "false",
		deprecatedAnnotation, "false")

	assert.True(t, propertyFunction(dk))

	// Default is false
	dk = createDynakubeWithAnnotation()

	assert.False(t, propertyFunction(dk))
}

func TestDeprecatedDisableAnnotations(t *testing.T) {
	t.Run(AnnotationFeatureActiveGateUpdates, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureActiveGateUpdates,
			AnnotationFeatureDisableActiveGateUpdates,
			func(dk DynaKube) bool {
				return dk.FeatureDisableActiveGateUpdates()
			})
	})
}

func TestDeprecatedEnableAnnotations(t *testing.T) {
	dk := createDynakubeWithAnnotation(AnnotationInjectionFailurePolicy, "fail")
	assert.Equal(t, "fail", dk.FeatureInjectionFailurePolicy())
}

func TestMaxMountAttempts(t *testing.T) {
	dk := createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "5")

	assert.Equal(t, 5, dk.FeatureMaxFailedCsiMountAttempts())

	dk = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "3")

	assert.Equal(t, 3, dk.FeatureMaxFailedCsiMountAttempts())

	dk = createDynakubeWithAnnotation()

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dk.FeatureMaxFailedCsiMountAttempts())

	dk = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "a")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dk.FeatureMaxFailedCsiMountAttempts())

	dk = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "-5")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dk.FeatureMaxFailedCsiMountAttempts())
}

func TestDynaKube_FeatureIgnoredNamespaces(t *testing.T) {
	dk := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
	}
	ignoredNamespaces := dk.getDefaultIgnoredNamespaces()
	dynakubeNamespaceMatches := false

	for _, namespace := range ignoredNamespaces {
		regex, err := regexp.Compile(namespace)

		require.NoError(t, err)

		match := regex.MatchString(dk.Namespace)

		if match {
			dynakubeNamespaceMatches = true
		}
	}

	assert.True(t, dynakubeNamespaceMatches)
}

func TestDefaultEnabledFeatureFlags(t *testing.T) {
	dk := createDynakubeEmptyDynakube()

	assert.True(t, dk.FeatureAutomaticKubernetesApiMonitoring())
	assert.True(t, dk.FeatureAutomaticInjection())
	assert.Equal(t, "silent", dk.FeatureInjectionFailurePolicy())

	assert.False(t, dk.FeatureDisableActiveGateUpdates())
	assert.False(t, dk.FeatureLabelVersionDetection())
}

func TestInjectionFailurePolicy(t *testing.T) {
	dk := createDynakubeEmptyDynakube()

	modes := map[string]string{
		failPhrase:   failPhrase,
		silentPhrase: silentPhrase,
	}
	for configuredMode, expectedMode := range modes {
		t.Run(`injection failure policy: `+configuredMode, func(t *testing.T) {
			dk.Annotations[AnnotationInjectionFailurePolicy] = configuredMode

			assert.Equal(t, expectedMode, dk.FeatureInjectionFailurePolicy())
		})
	}
}

func TestAgentInitialConnectRetry(t *testing.T) {
	t.Run("default => not set", func(t *testing.T) {
		dk := createDynakubeEmptyDynakube()

		initialRetry := dk.FeatureAgentInitialConnectRetry()
		require.Equal(t, -1, initialRetry)
	})
	t.Run("istio default => set", func(t *testing.T) {
		dk := createDynakubeEmptyDynakube()
		dk.Spec.EnableIstio = true

		initialRetry := dk.FeatureAgentInitialConnectRetry()
		require.Equal(t, IstioDefaultOneAgentInitialConnectRetry, initialRetry)
	})
	t.Run("istio default can be overruled", func(t *testing.T) {
		dk := createDynakubeEmptyDynakube()
		dk.Spec.EnableIstio = true
		dk.Annotations[AnnotationFeatureOneAgentInitialConnectRetry] = "5"

		initialRetry := dk.FeatureAgentInitialConnectRetry()
		require.Equal(t, 5, initialRetry)
	})
}
