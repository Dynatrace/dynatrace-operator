// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcile(t *testing.T) {
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "prometheusmonitoring", Namespace: "dynatrace"}}

	assertReconcileDone := func(t *testing.T, r *Reconciler, req ctrl.Request) {
		t.Helper()
		result, err := r.Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Empty(t, result)
	}

	t.Run("get prometheusmonitoring error", func(t *testing.T) {
		expectErr := k8serrors.NewInternalError(errors.New("BOOM"))
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return expectErr
			},
		}, &prometheusmonitoring.PrometheusMonitoring{})
		_, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("get prometheusmonitoring deleted", func(t *testing.T) {
		c := fake.NewClient(&prometheusmonitoring.PrometheusMonitoring{})
		assertReconcileDone(t, NewReconciler(c), req)
	})

	t.Run("get dynakube error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		expectErr := k8serrors.NewInternalError(errors.New("BOOM"))
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if out, ok := obj.(*prometheusmonitoring.PrometheusMonitoring); ok {
					pm.DeepCopyInto(out)

					return nil
				}

				return expectErr
			},
		}, pm)
		_, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("dynakube not found", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		c := fake.NewClient(pm)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Deploying, pm.Status.Phase)
	})

	t.Run("dynakube not running", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}}
		c := fake.NewClient(pm, dk)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Deploying, pm.Status.Phase)
	})

	t.Run("no token secret", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		c := fake.NewClient(pm, dk)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})

	t.Run("data-ingest token missing from secret", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Data: map[string][]byte{token.APIKey: []byte("api-token")}}
		c := fake.NewClient(pm, dk, secret)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})

	t.Run("build client error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		expectErr := errors.New("boom")
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
			return nil, expectErr
		}

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})

	t.Run("target allocator error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}

		expectErr := errors.New("boom")
		gm := newMockGatewayReconciler(t)
		gm.EXPECT().Reconcile(t.Context(), pm, dk, image.Client(nil)).Return(nil).Once()
		m := newMockTargetAllocatorReconciler(t)
		m.EXPECT().Reconcile(t.Context(), pm, dk, image.Client(nil)).Return(expectErr).Once()
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
			return &dynatrace.Client{}, nil
		}
		r.gateway = gm
		r.targetAllocator = m

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})

	t.Run("scraper error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}

		expectErr := errors.New("boom")
		gm := newMockGatewayReconciler(t)
		gm.EXPECT().Reconcile(t.Context(), pm, dk, image.Client(nil)).Return(nil).Once()
		m := newMockTargetAllocatorReconciler(t)
		m.EXPECT().Reconcile(t.Context(), pm, dk, image.Client(nil)).Return(nil).Once()
		sm := newMockScraperReconciler(t)
		sm.EXPECT().Reconcile(t.Context(), pm, dk, image.Client(nil)).Return(expectErr).Once()
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
			return &dynatrace.Client{}, nil
		}
		r.gateway = gm
		r.targetAllocator = m
		r.scraper = sm

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})
}

func Test_setPhase(t *testing.T) {
	conditionTrue := metav1.Condition{Type: "Available", Status: metav1.ConditionTrue, Reason: status.ReasonAvailable}
	conditionReconciling := metav1.Condition{Type: "Available", Status: metav1.ConditionFalse, Reason: status.ReasonReconciling}
	conditionError := metav1.Condition{Type: "Available", Status: metav1.ConditionFalse, Reason: status.ReasonError}

	boom := errors.New("boom")

	tests := []struct {
		name          string
		err           error
		conditions    []metav1.Condition
		expectedPhase status.DeploymentPhase
		expectedErr   error
	}{
		{"missing dynakube on creation", errDynaKubeNotFound, nil, status.Deploying, nil},
		{"missing dynakube after creation", errDynaKubeNotFound, []metav1.Condition{conditionTrue}, status.Error, nil},
		{"not ready dynakube on creation", errDynaKubeNotReady, nil, status.Deploying, nil},
		{"not ready dynakube after creation", errDynaKubeNotReady, []metav1.Condition{conditionTrue}, status.Deploying, nil},
		{"data-ingest token unavailable", errDataIngestTokenUnavailable, nil, status.Error, nil},
		{"generic error without conditions", boom, nil, status.Error, boom},
		{"no error without conditions", nil, nil, status.Deploying, nil},
		{"no error with all conditions true", nil, []metav1.Condition{conditionTrue, conditionTrue, conditionTrue}, status.Running, nil},
		{"no error with reconciling condition", nil, []metav1.Condition{conditionTrue, conditionReconciling, conditionTrue}, status.Deploying, nil},
		{"error condition has highest precedence", nil, []metav1.Condition{conditionTrue, conditionReconciling, conditionError}, status.Error, nil},
		{"generic error alongside healthy conditions preserves both the error and the running phase", boom, []metav1.Condition{conditionTrue}, status.Error, boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &prometheusmonitoring.PrometheusMonitoring{Status: prometheusmonitoring.PrometheusMonitoringStatus{Conditions: tt.conditions}}

			gotErr := setPhase(pm, tt.err)

			require.Equal(t, tt.expectedPhase, pm.Status.Phase)

			if tt.expectedErr == nil {
				require.NoError(t, gotErr)
			} else {
				require.ErrorIs(t, gotErr, tt.expectedErr)
			}
		})
	}
}

func Test_dynaKubeProxyChanged(t *testing.T) {
	tests := []struct {
		name     string
		oldProxy *value.Source
		newProxy *value.Source
		expect   bool
	}{
		{"both nil", nil, nil, false},
		{"old nil", nil, &value.Source{Value: "test"}, true},
		{"new nil", &value.Source{Value: "test"}, nil, true},
		{"value equal", &value.Source{Value: "test"}, &value.Source{Value: "test"}, false},
		{"value diff", &value.Source{Value: "foo"}, &value.Source{Value: "bar"}, true},
		{"valueFrom equal", &value.Source{ValueFrom: "test"}, &value.Source{ValueFrom: "test"}, false},
		{"valueFrom diff", &value.Source{ValueFrom: "foo"}, &value.Source{ValueFrom: "bar"}, true},
		{"field diff", &value.Source{Value: "test"}, &value.Source{ValueFrom: "test"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dynaKubeProxyChanged(
				&dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Proxy: tt.oldProxy}},
				&dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Proxy: tt.newProxy}},
			)
			assert.Equal(t, tt.expect, got)
		})
	}
}
