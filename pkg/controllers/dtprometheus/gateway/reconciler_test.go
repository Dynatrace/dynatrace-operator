// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// The condition branch logic is covered by the condition package. What matters here is
// that the gateway wires itself to the right condition type, component name and rollout check.
func TestReconcileCondition(t *testing.T) {
	t.Run("freshly created statefulset with no ready replicas -> pending", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Gateway.Image = "registry.example.com/gateway:1.2.3"
		dtp.Spec.Gateway.Replicas = new(int32(2))

		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"

		r := &Reconciler{Client: fake.NewClient()}
		require.NoError(t, r.Reconcile(t.Context(), dtp, dk, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.GatewayAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonReconciling, condition.Reason)
		assert.Equal(t, "gateway is pending", condition.Message)
	})

	t.Run("reconcile error -> error, with unwrapped message", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Gateway.Image = "registry.example.com/gateway:1.2.3"

		boom := errors.New("boom")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
				return boom
			},
		})

		r := &Reconciler{Client: clt}
		require.Error(t, r.Reconcile(t.Context(), dtp, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.GatewayAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonError, condition.Reason)
		assert.Equal(t, boom.Error(), condition.Message)
	})
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
