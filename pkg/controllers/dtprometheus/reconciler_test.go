// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
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
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "dtprometheus", Namespace: "dynatrace"}}

	assertReconcileDone := func(t *testing.T, r *Reconciler, req ctrl.Request) {
		t.Helper()
		result, err := r.Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Empty(t, result)
	}

	t.Run("get dtprometheus error", func(t *testing.T) {
		expectErr := k8serrors.NewInternalError(errors.New("BOOM"))
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return expectErr
			},
		}, &dtprometheus.DTPrometheus{})
		_, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("get dtprometheus deleted", func(t *testing.T) {
		c := fake.NewClient(&dtprometheus.DTPrometheus{})
		assertReconcileDone(t, NewReconciler(c), req)
	})

	t.Run("get dynakube error", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		expectErr := k8serrors.NewInternalError(errors.New("BOOM"))
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if out, ok := obj.(*dtprometheus.DTPrometheus); ok {
					dtp.DeepCopyInto(out)

					return nil
				}

				return expectErr
			},
		}, dtp)
		_, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("dynakube not found", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		c := fake.NewClient(dtp)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp))
		require.Equal(t, status.Deploying, dtp.Status.Phase)
	})

	t.Run("dynakube not running", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}}
		c := fake.NewClient(dtp, dk)
		assertReconcileDone(t, NewReconciler(c), req)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp))
		require.Equal(t, status.Deploying, dtp.Status.Phase)
	})

	t.Run("no token secret", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		c := fake.NewClient(dtp, dk)
		_, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.Error(t, err)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp))
		require.Equal(t, status.Error, dtp.Status.Phase)
	})

	t.Run("build client error", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, StringData: map[string]string{token.APIKey: "api-token"}}
		c := fake.NewClient(dtp, dk, secret)
		r := NewReconciler(c)
		expectErr := errors.New("boom")
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
			return nil, expectErr
		}

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp))
		require.Equal(t, status.Error, dtp.Status.Phase)
	})

	t.Run("target allocator error", func(t *testing.T) {
		dtp := &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, Spec: dtprometheus.DTPrometheusSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, Status: dynakube.DynaKubeStatus{Phase: status.Running}}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: req.Namespace}, StringData: map[string]string{token.APIKey: "api-token"}}

		expectErr := errors.New("boom")
		gm := newMockGatewayReconciler(t)
		gm.EXPECT().Reconcile(t.Context(), dtp, dk, image.Client(nil)).Return(nil).Once()
		m := newMockTargetAllocatorReconciler(t)
		m.EXPECT().Reconcile(t.Context(), dtp, dk, image.Client(nil)).Return(expectErr).Once()
		c := fake.NewClient(dtp, dk, secret)
		r := NewReconciler(c)
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
			return &dynatrace.Client{}, nil
		}
		r.gateway = gm
		r.targetAllocator = m

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp))
		require.Equal(t, status.Error, dtp.Status.Phase)
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
		{"generic error without conditions", boom, nil, status.Error, boom},
		{"no error without conditions", nil, nil, status.Deploying, nil},
		{"no error with all conditions true", nil, []metav1.Condition{conditionTrue, conditionTrue, conditionTrue}, status.Running, nil},
		{"no error with reconciling condition", nil, []metav1.Condition{conditionTrue, conditionReconciling, conditionTrue}, status.Deploying, nil},
		{"error condition has highest precedence", nil, []metav1.Condition{conditionTrue, conditionReconciling, conditionError}, status.Error, nil},
		{"generic error alongside healthy conditions preserves both the error and the running phase", boom, []metav1.Condition{conditionTrue}, status.Error, boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &dtprometheus.DTPrometheus{Status: dtprometheus.DTPrometheusStatus{Conditions: tt.conditions}}

			gotErr := setPhase(dtp, tt.err)

			require.Equal(t, tt.expectedPhase, dtp.Status.Phase)

			if tt.expectedErr == nil {
				require.NoError(t, gotErr)
			} else {
				require.ErrorIs(t, gotErr, tt.expectedErr)
			}
		})
	}
}
