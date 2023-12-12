package dynakube

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func createDynakubeEmptyDynakube() DynaKube {
	dynakube := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
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
	t.Run(AnnotationFeatureMetadataEnrichment, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			AnnotationFeatureMetadataEnrichment,
			AnnotationFeatureDisableMetadataEnrichment,
			func(dynakube DynaKube) bool {
				return dynakube.FeatureDisableMetadataEnrichment()
			})
	})
}

func TestDeprecatedEnableAnnotations(t *testing.T) {
	// New annotation works
	dynakube := createDynakubeWithAnnotation(AnnotationFeatureActiveGateAuthToken, "false")

	assert.False(t, dynakube.FeatureActiveGateAuthToken())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureActiveGateAuthToken, "true")

	assert.True(t, dynakube.FeatureActiveGateAuthToken())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureActiveGateAuthToken, "false")

	assert.False(t, dynakube.FeatureActiveGateAuthToken())

	// Default is true
	dynakube = createDynakubeWithAnnotation()
	assert.True(t, dynakube.FeatureActiveGateAuthToken())
}

func TestMaxMountAttempts(t *testing.T) {
	dynakube := createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "5")

	assert.Equal(t, 5, dynakube.FeatureMaxFailedCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "3")

	assert.Equal(t, 3, dynakube.FeatureMaxFailedCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation()

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dynakube.FeatureMaxFailedCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "a")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dynakube.FeatureMaxFailedCsiMountAttempts())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureMaxFailedCsiMountAttempts, "-5")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, dynakube.FeatureMaxFailedCsiMountAttempts())
}

func TestDynaKube_FeatureIgnoredNamespaces(t *testing.T) {
	dynakube := DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		},
	}
	ignoredNamespaces := dynakube.getDefaultIgnoredNamespaces()
	dynakubeNamespaceMatches := false

	for _, namespace := range ignoredNamespaces {
		regex, err := regexp.Compile(namespace)

		require.NoError(t, err)

		match := regex.MatchString(dynakube.Namespace)

		if match {
			dynakubeNamespaceMatches = true
		}
	}

	assert.True(t, dynakubeNamespaceMatches)
}

func TestSyntheticMonitoringFlags(t *testing.T) {
	t.Run("with non empty loc id", func(t *testing.T) {
		const locOrdinal = uint64(77777777777)
		locId := fmt.Sprintf("SYNTHETIC_LOCATION-%x", locOrdinal)
		dynaKube := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureSyntheticLocationEntityId: locId,
				},
			},
		}
		assert.Equal(t,
			locId,
			dynaKube.FeatureSyntheticLocationEntityId(),
			"declared syn loc entity id: %s",
			locId)
	})

	t.Run("with default node type", func(t *testing.T) {
		dynaKube := DynaKube{}
		assert.Equal(t,
			SyntheticNodeS,
			dynaKube.FeatureSyntheticNodeType(),
			"default node type: %s",
			SyntheticNodeS)
	})

	t.Run("with declared node type", func(t *testing.T) {
		dynaKube := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureSyntheticNodeType: SyntheticNodeXs,
				},
			},
		}
		assert.Equal(t,
			SyntheticNodeXs,
			dynaKube.FeatureSyntheticNodeType(),
			"declared node type: %s",
			SyntheticNodeXs)
	})

	t.Run("with default replicas", func(t *testing.T) {
		dynaKube := DynaKube{}
		assert.Equal(t,
			defaultSyntheticReplicas,
			dynaKube.FeatureSyntheticReplicas(),
			"default replicas: %s",
			defaultSyntheticReplicas)
	})

	t.Run("with declared replicas", func(t *testing.T) {
		replicas := int32(7)
		dynaKube := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureSyntheticReplicas: strconv.Itoa(int(replicas)),
				},
			},
		}
		assert.Equal(t,
			replicas,
			dynaKube.FeatureSyntheticReplicas(),
			"declared replicas: %s",
			replicas)
	})
}

func TestDefaultEnabledFeatureFlags(t *testing.T) {
	dynakube := createDynakubeEmptyDynakube()

	assert.True(t, dynakube.FeatureActiveGateAuthToken())
	assert.True(t, dynakube.FeatureActiveGateReadOnlyFilesystem())
	assert.True(t, dynakube.FeatureAutomaticKubernetesApiMonitoring())
	assert.True(t, dynakube.FeatureAutomaticInjection())
	assert.True(t, dynakube.FeatureInjectionFailurePolicy() == "silent")

	assert.False(t, dynakube.FeatureDisableActiveGateUpdates())
	assert.False(t, dynakube.FeatureDisableMetadataEnrichment())
	assert.False(t, dynakube.FeatureLabelVersionDetection())
}

func TestInjectionFailurePolicy(t *testing.T) {
	dynakube := createDynakubeEmptyDynakube()

	modes := map[string]string{
		failPhrase:   failPhrase,
		silentPhrase: silentPhrase,
	}
	for configuredMode, expectedMode := range modes {
		t.Run(`injection failure policy: `+configuredMode, func(t *testing.T) {
			dynakube.Annotations[AnnotationInjectionFailurePolicy] = configuredMode

			assert.Equal(t, expectedMode, dynakube.FeatureInjectionFailurePolicy())
		})
	}
}

func TestAgentInitialConnectRetry(t *testing.T) {
	t.Run("default => not set", func(t *testing.T) {
		dynakube := createDynakubeEmptyDynakube()

		initialRetry := dynakube.FeatureAgentInitialConnectRetry()
		require.Equal(t, -1, initialRetry)
	})
	t.Run("istio default => set", func(t *testing.T) {
		dynakube := createDynakubeEmptyDynakube()
		dynakube.Spec.EnableIstio = true

		initialRetry := dynakube.FeatureAgentInitialConnectRetry()
		require.Equal(t, IstioDefaultOneAgentInitialConnectRetry, initialRetry)
	})
	t.Run("istio default can be overruled", func(t *testing.T) {
		dynakube := createDynakubeEmptyDynakube()
		dynakube.Spec.EnableIstio = true
		dynakube.Annotations[AnnotationFeatureOneAgentInitialConnectRetry] = "5"

		initialRetry := dynakube.FeatureAgentInitialConnectRetry()
		require.Equal(t, 5, initialRetry)
	})
}
