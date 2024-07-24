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
			createDeployment(), testStatefulSetName)
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Empty(t, service.Labels)
	})
	t.Run("create StatefulSet with labels", func(t *testing.T) {
		labelName := "name"
		labelValue := "value"
		labels := map[string]string{
			labelName: labelValue,
		}
		appLabels := map[string]string{
			"version": "1.0.0",
		}
		service, err := Build(
			createDeployment(), testStatefulSetName, SetAllLabels(labels, appLabels))
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		require.Len(t, service.Labels, 1)
		require.Len(t, service.Spec.Selector.MatchLabels, 1)
		// only Template labels must have both since we merge them
		require.Len(t, service.Spec.Template.ObjectMeta.Labels, 2)
		assert.Equal(t, labelValue, service.Labels[labelName])
		assert.Equal(t, labelValue, service.Spec.Template.ObjectMeta.Labels[labelName])
		assert.Equal(t, "1.0.0", service.Spec.Template.ObjectMeta.Labels["version"])
	})
}
