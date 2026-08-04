// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDTPrometheusSelfMonitoringDefaulting(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	clt.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceDtp,
			Labels: map[string]string{},
		},
	})

	t.Run("key entirely absent from the manifest defaults to an empty (enabled) object", func(t *testing.T) {
		raw := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "dynatrace.com/v1alpha1",
			"kind":       "DTPrometheus",
			"metadata": map[string]any{
				"name":      "selfmon-key-omitted",
				"namespace": testNamespaceDtp,
			},
			"spec": map[string]any{
				"dynaKubeName": "dynakube",
			},
		}}

		require.NoError(t, clt.Create(t.Context(), raw))
		t.Cleanup(func() { assert.NoError(t, clt.Delete(context.Background(), raw)) })

		var dtp dtprometheus.DTPrometheus
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: "selfmon-key-omitted", Namespace: testNamespaceDtp}, &dtp))

		require.NotNil(t, dtp.Spec.SelfMonitoring)
		assert.True(t, dtp.IsSelfMonitoringEnabled())
	})

	t.Run("selfMonitoring explicitly set to null stays disabled", func(t *testing.T) {
		raw := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "dynatrace.com/v1alpha1",
			"kind":       "DTPrometheus",
			"metadata": map[string]any{
				"name":      "selfmon-null",
				"namespace": testNamespaceDtp,
			},
			"spec": map[string]any{
				"dynaKubeName":   "dynakube",
				"selfMonitoring": nil,
			},
		}}

		require.NoError(t, clt.Create(t.Context(), raw))
		t.Cleanup(func() { assert.NoError(t, clt.Delete(context.Background(), raw)) })

		var dtp dtprometheus.DTPrometheus
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: "selfmon-null", Namespace: testNamespaceDtp}, &dtp))

		assert.Nil(t, dtp.Spec.SelfMonitoring)
		assert.False(t, dtp.IsSelfMonitoringEnabled())
	})

	t.Run("typed client with a zero-value spec serializes explicit null and stays disabled", func(t *testing.T) {
		// SelfMonitoring has no omitzero json tag, so a nil *SelfMonitoringSpec on a
		// Go-constructed object always marshals as an explicit "null", not an absent
		// key. Only a manifest that omits the key outright (see the case above) picks
		// up the CRD default.
		dtp := buildDTPrometheus()
		dtp.Name = "selfmon-typed-zero-value"

		require.NoError(t, clt.Create(t.Context(), dtp))
		t.Cleanup(func() { assert.NoError(t, clt.Delete(context.Background(), dtp)) })

		assert.Nil(t, dtp.Spec.SelfMonitoring)
		assert.False(t, dtp.IsSelfMonitoringEnabled())
	})
}
