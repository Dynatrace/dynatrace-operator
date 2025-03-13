package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddVersionDetectionEnvs(t *testing.T) {
	const (
		customVersionValue               = "my awesome custom version"
		customProductValue               = "my awesome custom product"
		customReleaseStageValue          = "my awesome custom stage"
		customBuildVersionValue          = "my awesome custom build version"
		customVersionAnnotationName      = "custom-version"
		customProductAnnotationName      = "custom-product"
		customStageAnnotationName        = "custom-stage"
		customBuildVersionAnnotationName = "custom-build-version"
		customVersionFieldPath           = "metadata.podAnnotations['" + customVersionAnnotationName + "']"
		customProductFieldPath           = "metadata.podAnnotations['" + customProductAnnotationName + "']"
		customStageFieldPath             = "metadata.podAnnotations['" + customStageAnnotationName + "']"
		customBuildVersionFieldPath      = "metadata.podAnnotations['" + customBuildVersionAnnotationName + "']"
	)

	t.Run("version and product env vars are set using values referenced in namespace podAnnotations", func(t *testing.T) {
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
			productMappingAnnotationName: customProductFieldPath,
		}
		expectedMappings := map[string]string{
			ReleaseVersionEnv: customVersionFieldPath,
			ReleaseProductEnv: customProductFieldPath,
		}
		unexpectedMappingsKeys := []string{ReleaseStageEnv, ReleaseBuildVersionEnv}

		doTestMappings(t, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
	})
	t.Run("only version env vars is set using value referenced in namespace podAnnotations, product is default", func(t *testing.T) {
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
		}
		expectedMappings := map[string]string{
			ReleaseVersionEnv: customVersionFieldPath,
			ReleaseProductEnv: defaultVersionLabelMapping[ReleaseProductEnv],
		}
		unexpectedMappingsKeys := []string{ReleaseStageEnv, ReleaseBuildVersionEnv}

		doTestMappings(t, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
	})
	t.Run("optional env vars (stage, build-version) are set using values referenced in namespace podAnnotations, default ones remain default", func(t *testing.T) {
		namespaceAnnotations := map[string]string{
			stageMappingAnnotationName: customStageFieldPath,
			buildVersionAnnotationName: customBuildVersionFieldPath,
		}
		expectedMappings := map[string]string{
			ReleaseVersionEnv:      defaultVersionLabelMapping[ReleaseVersionEnv],
			ReleaseProductEnv:      defaultVersionLabelMapping[ReleaseProductEnv],
			ReleaseStageEnv:        customStageFieldPath,
			ReleaseBuildVersionEnv: customBuildVersionFieldPath,
		}

		doTestMappings(t, namespaceAnnotations, expectedMappings, nil)
	})
	t.Run("all env vars are namespace-podAnnotations driven", func(t *testing.T) {
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
			productMappingAnnotationName: customProductFieldPath,
			stageMappingAnnotationName:   customStageFieldPath,
			buildVersionAnnotationName:   customBuildVersionFieldPath,
		}
		expectedMappings := map[string]string{
			ReleaseVersionEnv:      customVersionFieldPath,
			ReleaseProductEnv:      customProductFieldPath,
			ReleaseStageEnv:        customStageFieldPath,
			ReleaseBuildVersionEnv: customBuildVersionFieldPath,
		}

		doTestMappings(t, namespaceAnnotations, expectedMappings, nil)
	})
}

func doTestMappings(t *testing.T, namespaceAnnotations map[string]string, expectedMappings map[string]string, unexpectedMappingsKeys []string) {
	container := corev1.Container{}

	AddVersionDetectionEnvs(&container, getTestNamespace(namespaceAnnotations))

	assertContainsMappings(t, expectedMappings, container)
	assertNotContainsMappings(t, unexpectedMappingsKeys, container)
}

func assertContainsMappings(t *testing.T, expectedMappings map[string]string, container corev1.Container) {
	for envName, fieldPath := range expectedMappings {
		assert.Contains(t, container.Env, corev1.EnvVar{
			Name: envName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "",
					FieldPath:  fieldPath,
				},
			},
		})
	}
}

func assertNotContainsMappings(t *testing.T, unexpectedMappingKeys []string, container corev1.Container) {
	for _, env := range container.Env {
		assert.NotContains(t, unexpectedMappingKeys, env.Name)
	}
}

func getTestNamespace(annotations map[string]string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-ns",
			Labels:      map[string]string{},
			Annotations: annotations,
		},
	}
}
