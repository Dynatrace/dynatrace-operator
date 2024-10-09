package deploymentmetadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func createTestDynakubeObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testName,
	}
}

func createTestDynakube(spec *dynakube.DynaKubeSpec) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{ObjectMeta: createTestDynakubeObjectMeta()}
	if spec != nil {
		dk.Spec = *spec
	}

	return dk
}

func TestReconcile(t *testing.T) {
	clusterID := "test"

	t.Run(`don't create anything, if no mode is configured`, func(t *testing.T) {
		dk := createTestDynakube(nil)
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(fakeClient, fakeClient, *dk, clusterID)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actualConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: GetDeploymentMetadataConfigMapName(testName), Namespace: testNamespace}, &actualConfigMap)
		require.Error(t, err)
	})
	t.Run(`delete configmap, if no mode is configured`, func(t *testing.T) {
		dk := createTestDynakube(nil)
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetDeploymentMetadataConfigMapName(testName),
					Namespace: testNamespace,
				},
			},
		).Build()
		r := NewReconciler(fakeClient, fakeClient, *dk, clusterID)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actualConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: GetDeploymentMetadataConfigMapName(testName), Namespace: testNamespace}, &actualConfigMap)
		require.Error(t, err)
	})

	t.Run(`create configmap with 1 key, if only oneagent is needed`, func(t *testing.T) {
		dk := createTestDynakube(
			&dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			})

		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(fakeClient, fakeClient, *dk, clusterID)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actualConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: GetDeploymentMetadataConfigMapName(testName), Namespace: testNamespace}, &actualConfigMap)
		require.NoError(t, err)
		require.NotEmpty(t, actualConfigMap.Data)
		assert.NotEmpty(t, actualConfigMap.Data[OneAgentMetadataKey])
	})

	t.Run(`create configmap with 1 key, if only activegate is needed`, func(t *testing.T) {
		dk := createTestDynakube(
			&dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			})

		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(fakeClient, fakeClient, *dk, clusterID)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actualConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: GetDeploymentMetadataConfigMapName(testName), Namespace: testNamespace}, &actualConfigMap)
		require.NoError(t, err)
		require.NotEmpty(t, actualConfigMap.Data)
		assert.NotEmpty(t, actualConfigMap.Data[ActiveGateMetadataKey])
	})
	t.Run(`create configmap with 2 keys, if both oneagent and activegate is needed`, func(t *testing.T) {
		dk := createTestDynakube(
			&dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			})

		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(fakeClient, fakeClient, *dk, clusterID)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actualConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: GetDeploymentMetadataConfigMapName(testName), Namespace: testNamespace}, &actualConfigMap)
		require.NoError(t, err)
		require.NotEmpty(t, actualConfigMap.Data)
		assert.NotEmpty(t, actualConfigMap.Data[OneAgentMetadataKey])
		assert.NotEmpty(t, actualConfigMap.Data[ActiveGateMetadataKey])
	})
}
