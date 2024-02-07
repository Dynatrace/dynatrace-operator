package configmap

import (
	"context"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
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
)

var configMapLog = logger.Get().WithName("test-configMap")

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestConfigMapQuery(t *testing.T) {
	t.Run(`Get configMap`, testGetConfigMap)
	t.Run(`Create configMap`, testCreateConfigMap)
	t.Run(`Update configMap`, testUpdateConfigMap)
	t.Run(`Create or update configMap`, testCreateOrUpdateConfigMap)
	t.Run(`Identical configMap is not updated`, testIdenticalConfigMapIsNotUpdated)
	t.Run(`Update configMap when data has changed`, testUpdateConfigMapWhenDataChanged)
	t.Run(`Update configMap when labels have changed`, testUpdateConfigMapWhenLabelsChanged)
	t.Run(`Create configMap in target namespace`, testCreateConfigMapInTargetNamespace)
	t.Run(`Delete configMap in target namespace`, testDeleteConfigMap)
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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	foundConfigMap, err := configMapQuery.Get(client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace})

	require.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, foundConfigMap))
}

func testCreateConfigMap(t *testing.T) {
	fakeClient := fake.NewClient()
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}

	err := configMapQuery.Create(configMap)

	require.NoError(t, err)

	var actualConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace}, &actualConfigMap)

	require.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, actualConfigMap))
}

func testUpdateConfigMap(t *testing.T) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}
	fakeClient := fake.NewClient()
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Update(configMap)

	require.Error(t, err)

	configMap.Data = nil
	fakeClient = fake.NewClient(&configMap)
	configMapQuery.KubeClient = fakeClient

	err = configMapQuery.Update(configMap)

	require.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, updatedConfigMap))
}

func testCreateOrUpdateConfigMap(t *testing.T) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}
	fakeClient := fake.NewClient()
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(configMap)
	require.NoError(t, err)

	var createdConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &createdConfigMap)

	require.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, createdConfigMap))

	fakeClient = fake.NewClient(&configMap)
	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: nil,
	}
	configMapQuery.KubeClient = fakeClient

	err = configMapQuery.CreateOrUpdate(configMap)

	require.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	require.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, updatedConfigMap))
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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	require.NoError(t, err)
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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	require.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	require.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)

	require.NoError(t, err)

	var newConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &newConfigMap)

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
	configMapQuery := NewQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Delete(*configMap)
	require.NoError(t, err)

	var deletedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &deletedConfigMap)
	require.Error(t, err)
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

func TestConfigMapBuilder(t *testing.T) {
	dataKey := "cfg"
	data := map[string]string{
		dataKey: "",
	}

	t.Run("create config map", func(t *testing.T) {
		configMap, err := CreateConfigMap(scheme.Scheme, createDeployment(),
			NewModifier(testConfigMapName),
			NewNamespaceModifier(testNamespace))
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, testConfigMapName, configMap.Name)
		assert.Empty(t, configMap.Data)
	})
	t.Run("create config map with data", func(t *testing.T) {
		configMap, err := CreateConfigMap(scheme.Scheme, createDeployment(),
			NewModifier(testConfigMapName),
			NewNamespaceModifier(testNamespace),
			NewConfigMapDataModifier(data))
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, testConfigMapName, configMap.Name)
		_, found := configMap.Data[dataKey]
		assert.True(t, found)
	})
}
