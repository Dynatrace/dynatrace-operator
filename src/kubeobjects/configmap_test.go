package kubeobjects

import (
	"context"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"
)

var configMapLog = logger.Factory.GetLogger("test-configMap")

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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	foundConfigMap, err := configMapQuery.Get(client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace})

	assert.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, foundConfigMap))
}

func testCreateConfigMap(t *testing.T) {
	fakeClient := fake.NewClient()
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{testKey1: testConfigMapValue},
	}

	err := configMapQuery.Create(configMap)

	assert.NoError(t, err)

	var actualConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testConfigMapName, Namespace: testNamespace}, &actualConfigMap)

	assert.NoError(t, err)
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Update(configMap)

	assert.Error(t, err)

	configMap.Data = nil
	fakeClient = fake.NewClient(&configMap)
	configMapQuery.kubeClient = fakeClient

	err = configMapQuery.Update(configMap)

	assert.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	assert.NoError(t, err)
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(configMap)
	assert.NoError(t, err)

	var createdConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &createdConfigMap)

	assert.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, createdConfigMap))

	fakeClient = fake.NewClient(&configMap)
	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: nil,
	}
	configMapQuery.kubeClient = fakeClient

	err = configMapQuery.CreateOrUpdate(configMap)

	assert.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, &updatedConfigMap)

	assert.NoError(t, err)
	assert.True(t, AreConfigMapsEqual(configMap, updatedConfigMap))
}

func testIdenticalConfigMapIsNotUpdated(t *testing.T) {
	data := map[string]string{testKey1: string(testValue1)}
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	assert.NoError(t, err)
}

func testUpdateConfigMapWhenDataChanged(t *testing.T) {
	data := map[string]string{testKey1: string(testValue1)}
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	assert.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, updatedConfigMap.Data))
}

func testUpdateConfigMapWhenLabelsChanged(t *testing.T) {
	data := map[string]string{testKey1: string(testValue1)}
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)
	assert.NoError(t, err)

	var updatedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &updatedConfigMap)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(labels, updatedConfigMap.Labels))
}

func testCreateConfigMapInTargetNamespace(t *testing.T) {
	data := map[string]string{testKey1: string(testValue1)}
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.CreateOrUpdate(*configMap)

	assert.NoError(t, err)

	var newConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &newConfigMap)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, newConfigMap.Data))
	assert.True(t, reflect.DeepEqual(labels, newConfigMap.Labels))
	assert.Equal(t, testConfigMapName, newConfigMap.Name)
	assert.Equal(t, testNamespace, newConfigMap.Namespace)
}

func testDeleteConfigMap(t *testing.T) {
	data := map[string]string{testKey1: string(testValue1)}
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
	configMapQuery := NewConfigMapQuery(context.TODO(), fakeClient, fakeClient, configMapLog)

	err := configMapQuery.Delete(*configMap)
	require.NoError(t, err)

	var deletedConfigMap corev1.ConfigMap
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testConfigMapName, Namespace: testNamespace}, &deletedConfigMap)
	assert.Error(t, err)
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
			NewConfigMapNameModifier(testConfigMapName),
			NewConfigMapNamespaceModifier(testNamespace))
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, deploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, configMap.Name, testConfigMapName)
		assert.Len(t, configMap.Data, 0)
	})
	t.Run("create config map with data", func(t *testing.T) {
		configMap, err := CreateConfigMap(scheme.Scheme, createDeployment(),
			NewConfigMapNameModifier(testConfigMapName),
			NewConfigMapNamespaceModifier(testNamespace),
			NewConfigMapDataModifier(data))
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, deploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, configMap.Name, testConfigMapName)
		_, found := configMap.Data[dataKey]
		assert.True(t, found)
	})
}
