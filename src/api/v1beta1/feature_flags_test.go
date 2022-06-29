package v1beta1

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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

func TestActiveGateUpdates(t *testing.T) {
	dynakube := createDynakubeWithAnnotation(AnnotationFeatureActiveGateUpdates, "false")

	assert.True(t, dynakube.FeatureDisableActiveGateUpdates())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureActiveGateUpdates, "true")

	assert.False(t, dynakube.FeatureDisableActiveGateUpdates())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureDisableActiveGateUpdates, "true")

	assert.True(t, dynakube.FeatureDisableActiveGateUpdates())

	dynakube = createDynakubeWithAnnotation(AnnotationFeatureDisableActiveGateUpdates, "false")

	assert.False(t, dynakube.FeatureDisableActiveGateUpdates())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureActiveGateUpdates, "true",
		AnnotationFeatureDisableActiveGateUpdates, "true")

	assert.False(t, dynakube.FeatureDisableActiveGateUpdates())

	dynakube = createDynakubeWithAnnotation(
		AnnotationFeatureActiveGateUpdates, "false",
		AnnotationFeatureDisableActiveGateUpdates, "false")

	assert.True(t, dynakube.FeatureDisableActiveGateUpdates())
}
