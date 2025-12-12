package configuration

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName  = "dynakube"
	testNamespaceName = "dynatrace"
)

func getTestDynakube(telemetryIngestSpec *telemetryingest.Spec) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynakube.DynaKubeSpec{
			TelemetryIngest: telemetryIngestSpec,
		},
		Status: dynakube.DynaKubeStatus{},
	}
}

func TestConfigurationConfigMap(t *testing.T) {
	t.Run("create configmap if it does not exist", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		configMap := &corev1.ConfigMap{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace}, configMap)
		require.NoError(t, err)

		_, ok := configMap.Data[consts.ConfigFieldName]
		assert.True(t, ok)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, conditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, k8sconditions.ConfigMapCreatedOrUpdatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	})
}
