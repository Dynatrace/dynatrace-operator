package k8sconfigmap

import (
	"context"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"
	testDeploymentName = "deployment-as-owner-of-secret"
	testValue1         = "test-value"
	testKey1           = "test-key"
	testNamespace      = "test-namespace"
	annotationHash     = api.InternalFlagPrefix + "template-hash"
)

var configMapLog = logd.Get().WithName("test-configMap")

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestConfigMapQuery(t *testing.T) {
	t.Run("Get configMap", testGetConfigMap)
	t.Run("Create configMap", testCreateConfigMap)
	t.Run("Update configMap", testUpdateConfigMap)
	t.Run("Create or update configMap", testCreateOrUpdateConfigMap)
	t.Run("Identical configMap is not updated", testIdenticalConfigMapIsNotUpdated)
	t.Run("Update configMap when data has changed", testUpdateConfigMapWhenDataChanged)
	t.Run("Update configMap when labels have changed", testUpdateConfigMapWhenLabelsChanged)
	t.Run("Create configMap in target namespace", testCreateConfigMapInTargetNamespace)
	t.Run("Delete configMap in target namespace", testDeleteConfigMap)
	t.Run("Hash annotation is there after create", testHashAnnotationAfterCreate)
}

func testGetConfigMap(t *testing.T) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}
	fakeClient := fake.NewClient(&configMap)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	foundConfigMap, err := configMapQuery.Get(context.Background(), client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace})

	require.NoError(t, err)
	assert.True(t, isEqual(&configMap, foundConfigMap))
}

func testCreateConfigMap(t *testing.T) {
	fakeClient := fake.NewClient()
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}

	err := configMapQuery.Create(context.Background(), configMap)

	require.NoError(t, err)

	var actualConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace}, &actualConfigMap)

	require.NoError(t, err)
	assert.True(t, isEqual(configMap, &actualConfigMap))
}

func testUpdateConfigMap(t *testing.T) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}
	fakeClient := fake.NewClient()
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Update(context.Background(), configMap)

	require.Error(t, err)

	configMap.Data = nil
	fakeClient = fake.NewClient(configMap)
	configMapQuery.KubeClient = fakeClient

	err = configMapQuery.Update(context.Background(), configMap)

	require.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, isEqual(configMap, &updatedConfigMap))
}

func testCreateOrUpdateConfigMap(t *testing.T) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}
	fakeClient := fake.NewClient()
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	created, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.True(t, created)

	var createdConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &createdConfigMap)

	require.NoError(t, err)
	assert.True(t, isEqual(configMap, &createdConfigMap))

	fakeClient = fake.NewClient(configMap)
	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: nil,
	}
	configMapQuery.KubeClient = fakeClient

	updated, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.True(t, updated)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, isEqual(configMap, &updatedConfigMap))
}

func testIdenticalConfigMapIsNotUpdated(t *testing.T) {
	data := map[string]string{testKey1: testValue1}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: data,
	})
	configMap := createTestConfigMap(labels, data)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	updated, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.False(t, updated)
}

func testUpdateConfigMapWhenDataChanged(t *testing.T) {
	data := map[string]string{testKey1: testValue1}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	})
	configMap := createTestConfigMap(labels, data)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	updated, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.True(t, updated)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, updatedConfigMap.Data))
}

func testUpdateConfigMapWhenLabelsChanged(t *testing.T) {
	data := map[string]string{testKey1: testValue1}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
			Labels:    map[string]string{},
		},
		Data: data,
	})
	configMap := createTestConfigMap(labels, data)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	updated, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.True(t, updated)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(labels, updatedConfigMap.Labels))
}

func testCreateConfigMapInTargetNamespace(t *testing.T) {
	data := map[string]string{testKey1: testValue1}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: "other",
		},
		Data: map[string]string{},
	})
	configMap := createTestConfigMap(labels, data)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	updated, err := configMapQuery.CreateOrUpdate(context.Background(), configMap)
	require.NoError(t, err)
	require.True(t, updated)

	var newConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &newConfigMap)

	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, newConfigMap.Data))
	assert.True(t, reflect.DeepEqual(labels, newConfigMap.Labels))
	assert.Equal(t, testConfigMapName, newConfigMap.Name)
	assert.Equal(t, testNamespace, newConfigMap.Namespace)
}

func testDeleteConfigMap(t *testing.T) {
	data := map[string]string{testKey1: testValue1}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: data,
	})
	configMap := createTestConfigMap(labels, data)
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Delete(context.Background(), configMap)
	require.NoError(t, err)

	var deletedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &deletedConfigMap)
	require.Error(t, err)
}

func testHashAnnotationAfterCreate(t *testing.T) {
	fakeClient := fake.NewClient()
	configMapQuery := Query(fakeClient, fakeClient, configMapLog)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}

	err := configMapQuery.Create(context.Background(), configMap)

	require.NoError(t, err)

	var actualConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace}, &actualConfigMap)

	require.NoError(t, err)
	assert.True(t, isEqual(configMap, &actualConfigMap))

	assert.NotEmpty(t, configMap.Annotations)
	assert.NotEmpty(t, configMap.Annotations[annotationHash])
	assert.NotEmpty(t, actualConfigMap.Annotations)
	assert.NotEmpty(t, actualConfigMap.Annotations[annotationHash])

	assert.Equal(t, configMap.Annotations[annotationHash], actualConfigMap.Annotations[annotationHash])
}

func createTestConfigMap(labels map[string]string, data map[string]string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: data,
	}

	return configMap
}
