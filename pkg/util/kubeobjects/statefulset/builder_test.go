package statefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDeploymentName  = "deployment-as-owner-of-service"
	testStatefulSetName = "test-statefulset-name"
	testNamespace       = "test-namespace"
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
		service, err := Build(
			createDeployment(), "test-name")
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Empty(t, service.Labels)
	})
}
