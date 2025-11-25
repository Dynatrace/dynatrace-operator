package k8sstatefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
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
	t.Run("create StatefulSet with labels", func(t *testing.T) {
		container := corev1.Container{}
		labelName := "name"
		labelValue := "value"
		customLabels := map[string]string{
			labelName: labelValue,
		}
		appLabels := k8slabel.NewAppLabels(testAppName, testDynakubeName, k8slabel.ExtensionComponentLabel, testVersion)
		statefulSet, err := Build(
			createDeployment(), testStatefulSetName, container, SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), customLabels))
		require.NoError(t, err)
		require.Len(t, statefulSet.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, statefulSet.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, statefulSet.Name)
		require.Len(t, statefulSet.Labels, 5)
		require.Len(t, statefulSet.Spec.Selector.MatchLabels, 3)
		// only Template labels must have both since we merge them
		require.Len(t, statefulSet.Spec.Template.Labels, 6)
		assert.Empty(t, statefulSet.Labels[labelName])
		assert.Equal(t, labelValue, statefulSet.Spec.Template.Labels[labelName])

		assert.Equal(t, testAppName, statefulSet.Labels[k8slabel.AppNameLabel])
		assert.Equal(t, testDynakubeName, statefulSet.Labels[k8slabel.AppCreatedByLabel])
		assert.Equal(t, version.AppName, statefulSet.Labels[k8slabel.AppManagedByLabel])
		assert.Equal(t, k8slabel.ExtensionComponentLabel, statefulSet.Labels[k8slabel.AppComponentLabel])
		assert.Equal(t, testVersion, statefulSet.Labels[k8slabel.AppVersionLabel])

		assert.Equal(t, testAppName, statefulSet.Spec.Template.Labels[k8slabel.AppNameLabel])
		assert.Equal(t, testDynakubeName, statefulSet.Spec.Template.Labels[k8slabel.AppCreatedByLabel])
		assert.Equal(t, version.AppName, statefulSet.Spec.Template.Labels[k8slabel.AppManagedByLabel])
		assert.Equal(t, k8slabel.ExtensionComponentLabel, statefulSet.Spec.Template.Labels[k8slabel.AppComponentLabel])
		assert.Equal(t, testVersion, statefulSet.Spec.Template.Labels[k8slabel.AppVersionLabel])
	})
}
