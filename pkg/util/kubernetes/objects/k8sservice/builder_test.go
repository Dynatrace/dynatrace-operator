package k8sservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDeploymentName = "deployment-as-owner-of-service"
	testServiceName    = "test-service-name"
	testNamespace      = "test-namespace"
)

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestServiceBuilder(t *testing.T) {
	labelName := "name"
	labelValue := "value"
	labels := map[string]string{
		labelName: labelValue,
	}

	t.Run("create service", func(t *testing.T) {
		service, err := Build(createDeployment(),
			testServiceName,
			labels,
			nil,
			setNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testServiceName, service.Name)
		assert.Empty(t, service.Labels)
	})
	t.Run("create service with label", func(t *testing.T) {
		secret, err := Build(createDeployment(),
			testServiceName,
			labels,
			nil,
			SetLabels(labels),
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testServiceName, secret.Name)
		require.Len(t, secret.Labels, 1)
		assert.Equal(t, labelValue, secret.Labels[labelName])
	})
}
