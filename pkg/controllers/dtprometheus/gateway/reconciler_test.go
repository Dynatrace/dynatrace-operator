// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

func newTestDTP(name, namespace string) *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("dtp-uid")}}
}

// assertGolden compares obj, marshaled to YAML, against testdata/name. The fake
// client is the only source of non-determinism (it stamps resourceVersion), so
// that field is cleared before comparing.
func assertGolden(t *testing.T, name string, obj client.Object) {
	t.Helper()

	obj.SetResourceVersion("")
	// Ensure object has GVK so that TypeMeta is included
	kinds, _, _ := scheme.Scheme.ObjectKinds(obj)
	require.Len(t, kinds, 1)
	obj.GetObjectKind().SetGroupVersionKind(kinds[0])

	got, err := yaml.Marshal(obj)
	require.NoError(t, err)

	want, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	assert.YAMLEq(t, string(want), string(got))
}

func newTestScope(dtp *dtprometheus.DTPrometheus) *reconcileScope {
	return newTestScopeWithDynaKube(dtp, &dynakube.DynaKube{})
}

func newTestScopeWithDynaKube(dtp *dtprometheus.DTPrometheus, dk *dynakube.DynaKube) *reconcileScope {
	return &reconcileScope{
		Owner:     dtp,
		DynaKube:  dk,
		Spec:      dtp.Gateway(),
		AppLabels: k8slabel.New("opentelemetry-gateway", "otel-gateway", ""),
	}
}

func TestReconcileCondition(t *testing.T) {
	boom := fmt.Errorf("wrap: %w", errors.New("boom"))

	completeStatefulSet := &appsv1.StatefulSet{
		Spec:   appsv1.StatefulSetSpec{Replicas: new(int32(2))},
		Status: appsv1.StatefulSetStatus{ReadyReplicas: 2},
	}

	tests := []struct {
		name        string
		err         error
		statefulSet *appsv1.StatefulSet
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{"statefulset not rolled out -> reconciling", nil, nil, metav1.ConditionFalse, status.ReasonReconciling, "gateway is pending"},
		{"rollout complete -> available", nil, completeStatefulSet, metav1.ConditionTrue, status.ReasonAvailable, "gateway is ready"},
		{"error -> error", boom, nil, metav1.ConditionFalse, status.ReasonError, "boom"},
		{"error takes precedence over complete rollout", boom, completeStatefulSet, metav1.ConditionFalse, status.ReasonError, "boom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dtp := newTestDTP("dtp", "dynatrace")
			s := &reconcileScope{Owner: dtp, StatefulSet: test.statefulSet}
			r := &Reconciler{}

			r.reconcileCondition(s, test.err)

			condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.GatewayAvailable)
			require.NotNil(t, condition)
			assert.Equal(t, test.wantStatus, condition.Status)
			assert.Equal(t, test.wantReason, condition.Reason)
			assert.Equal(t, test.wantMessage, condition.Message)
		})
	}
}

func TestReconcileConfigMap(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		s := newTestScopeWithDynaKube(dtp, dk)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, cm))

		assertGolden(t, "configmap.yaml", cm)

		sum := sha256.Sum256([]byte(cm.Data[gatewayConfigKey]))
		assert.Equal(t, hex.EncodeToString(sum[:]), s.ConfigMapHash)
	})

	t.Run("resource attributes rendered", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		dk.Spec.ResourceAttributes = map[string]string{"env": "prod"}
		s := newTestScopeWithDynaKube(dtp, dk)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, cm))

		assert.Contains(t, cm.Data[gatewayConfigKey], "resource/dynakube")
		assert.Contains(t, cm.Data[gatewayConfigKey], "key: env")
		assert.Contains(t, cm.Data[gatewayConfigKey], "value: prod")
	})

	t.Run("merge labels", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      s.Spec.GetStatefulSetName(),
			Namespace: dtp.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"},
		}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, cm))
		assert.Equal(t, "value", cm.Labels["custom"])
		assert.Equal(t, "otel-gateway", cm.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileConfigMap(t.Context(), newTestScope(newTestDTP("dtp", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconcileService(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, svc))

		assertGolden(t, "service.yaml", svc)
	})

	t.Run("merge labels", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		existing := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
			Name:      s.Spec.GetStatefulSetName(),
			Namespace: dtp.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"},
		}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, svc))
		assert.Equal(t, "value", svc.Labels["custom"])
		assert.Equal(t, "otel-gateway", svc.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileService(t.Context(), newTestScope(newTestDTP("dtp", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func createErrorClient(createErr error) client.Client {
	return fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, clt client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return createErr
		},
	})
}
