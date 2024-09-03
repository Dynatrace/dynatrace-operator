package statefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDeploymentName  = "deployment-as-owner-of-service"
	testStatefulSetName = "test-statefulset-name"
	testNamespace       = "test-namespace"
	testDynakubeName    = "dynakube"
	testVersion         = "1.0.0"
	testAppName         = "dynatrace-operator"
)

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestStatefulSetBuilder(t *testing.T) {
	t.Run("create StatefulSet", func(t *testing.T) {
		container := corev1.Container{}
		service, err := Build(
			createDeployment(), testStatefulSetName, container)
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Empty(t, service.Labels)
	})
	t.Run("create StatefulSet with basic container configuration", func(t *testing.T) {
		container := corev1.Container{Name: "nginx", Image: "nginx", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 80}}}
		service, err := Build(
			createDeployment(), testStatefulSetName, container)
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Len(t, service.Spec.Template.Spec.Containers, 1)
		assert.Equal(t, "nginx", service.Spec.Template.Spec.Containers[0].Name)
	})
}
