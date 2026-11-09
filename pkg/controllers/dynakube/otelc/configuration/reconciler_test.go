// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
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
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
		UID:       "dk-uid",
		Spec: dynakube.DynaKubeSpec{
			TelemetryIngest: telemetryIngestSpec,
		},
		Status: dynakube.DynaKubeStatus{},
	}
}

func TestReconcile(t *testing.T) {
	assertCondition := func(t *testing.T, dk *dynakube.DynaKube) {
		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, conditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, k8sconditions.ConfigMapCreatedOrUpdatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	}

	// the hash changes with every change of the config, which the golden already covers
	removeHash := func(cm *corev1.ConfigMap) {
		delete(cm.Annotations, hasher.AnnotationHash)
	}

	t.Run("without resource attributes", func(t *testing.T) {
		k8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{})
		err := NewReconciler(k8sClient, k8sClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		assertCondition(t, dk)

		cm := &corev1.ConfigMap{}
		err = k8sClient.Get(t.Context(), client.ObjectKey{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace}, cm)
		require.NoError(t, err)

		helpers.AssertGolden(t, "testdata/configmap.yaml", cm, removeHash)
	})

	t.Run("with resource attributes", func(t *testing.T) {
		k8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{})
		dk.Spec.ResourceAttributes = map[string]string{"team": "shopping-cart"}
		err := NewReconciler(k8sClient, k8sClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		assertCondition(t, dk)

		cm := &corev1.ConfigMap{}
		err = k8sClient.Get(t.Context(), client.ObjectKey{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace}, cm)
		require.NoError(t, err)

		helpers.AssertGolden(t, "testdata/configmap_with_resource_attributes.yaml", cm, removeHash)
	})

	for _, protocol := range otelcgen.RegisteredProtocols {
		t.Run("only protocol "+string(protocol), func(t *testing.T) {
			k8sClient := fake.NewFakeClient()
			dk := getTestDynakube(&telemetryingest.Spec{Protocols: []otelcgen.Protocol{protocol}})
			err := NewReconciler(k8sClient, k8sClient).Reconcile(t.Context(), dk)
			require.NoError(t, err)

			assertCondition(t, dk)

			cm := &corev1.ConfigMap{}
			err = k8sClient.Get(t.Context(), client.ObjectKey{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace}, cm)
			require.NoError(t, err)

			helpers.AssertGolden(t, "testdata/configmap_protocol_"+string(protocol)+".yaml", cm, removeHash)
		})
	}
}
