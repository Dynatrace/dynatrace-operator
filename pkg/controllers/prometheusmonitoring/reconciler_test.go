// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestReconcile(t *testing.T) {
	req := ctrl.Request{Name: "prometheusmonitoring", Namespace: "dynatrace"}

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
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
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
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
		c := fake.NewClient(pm)
		result, err := NewReconciler(c).Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, missingDependencyRequeueInterval, result.RequeueAfter)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)

		cond := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.DynaKubeAvailable)
		require.NotNil(t, cond)
		require.Equal(t, metav1.ConditionFalse, cond.Status)
		require.Equal(t, status.ReasonNotFound, cond.Reason)
		require.Equal(t, "The DynaKube 'dk' is not present in the 'dynatrace' namespace.", cond.Message)
	})

	t.Run("dynakube recovered after not found", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
		c := fake.NewClient(pm)
		r := NewReconciler(c)
		_, err := r.Reconcile(t.Context(), req)
		require.NoError(t, err)

		// DynaKube has no tenant URL, so reconcile stops at the prerequisites check.
		require.NoError(t, c.Create(t.Context(), &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace}))
		_, err = r.Reconcile(t.Context(), req)
		require.NoError(t, err)

		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		cond := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.DynaKubeAvailable)
		require.NotNil(t, cond)
		require.Equal(t, metav1.ConditionTrue, cond.Status)
		require.Equal(t, status.ReasonAvailable, cond.Reason)
	})

	t.Run("build client error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
		secret := &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		expectErr := errors.New("boom")
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
			return nil, expectErr
		}

		_, err := r.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, expectErr)
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
		require.Equal(t, status.Error, pm.Status.Phase)
	})

	t.Run("target allocator error", func(t *testing.T) {
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
		secret := &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}

		expectErr := errors.New("boom")
		gm := newMockGatewayReconciler(t)
		gm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
		m := newMockTargetAllocatorReconciler(t)
		m.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(expectErr).Once()
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
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
		pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
		dk := &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
		secret := &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}

		expectErr := errors.New("boom")
		gm := newMockGatewayReconciler(t)
		gm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
		m := newMockTargetAllocatorReconciler(t)
		m.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
		sm := newMockScraperReconciler(t)
		sm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(expectErr).Once()
		c := fake.NewClient(pm, dk, secret)
		r := NewReconciler(c)
		r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
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

func TestReconcile_prerequisites(t *testing.T) {
	req := ctrl.Request{Name: "prometheusmonitoring", Namespace: "dynatrace"}
	pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}

	newDK := func(apiURL string) *dynakube.DynaKube {
		return &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: apiURL}}
	}
	tokenSecret := func(data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: data}
	}

	tests := []struct {
		name    string
		objects []client.Object
		message string
	}{
		{"tenant url missing", []client.Object{newDK(""), tokenSecret(map[string][]byte{token.DataIngestKey: []byte("t")})}, errTenantURLMissing.Error()},
		{"token secret missing", []client.Object{newDK("https://tenant.dynatrace.com/api")}, errTokenSecretNotFound.Error() + " (dk)"},
		{"data-ingest token missing", []client.Object{newDK("https://tenant.dynatrace.com/api"), tokenSecret(map[string][]byte{token.APIKey: []byte("t")})}, errDataIngestTokenMissing.Error()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClient(append(tt.objects, pm.DeepCopy())...)

			result, err := NewReconciler(c).Reconcile(t.Context(), req)
			require.NoError(t, err)
			assert.Equal(t, missingDependencyRequeueInterval, result.RequeueAfter)

			got := &prometheusmonitoring.PrometheusMonitoring{}
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), got))
			assert.Equal(t, status.Error, got.Status.Phase)

			cond := meta.FindStatusCondition(got.Status.Conditions, prometheusmonitoring.PrerequisitesMet)
			require.NotNil(t, cond)
			assert.Equal(t, metav1.ConditionFalse, cond.Status)
			assert.Equal(t, status.ReasonError, cond.Reason)
			assert.Contains(t, cond.Message, tt.message)
		})
	}
}

func TestReconcile_prerequisitesReadError(t *testing.T) {
	req := ctrl.Request{Name: "prometheusmonitoring", Namespace: "dynatrace"}
	pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
	dk := &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
	forbidden := k8serrors.NewForbidden(corev1.Resource("secrets"), "dk", errors.New("BOOM"))

	c := fake.NewClientWithInterceptors(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				return forbidden
			}

			return c.Get(ctx, key, obj, opts...)
		},
	}, pm, dk)

	_, err := NewReconciler(c).Reconcile(t.Context(), req)

	require.ErrorIs(t, err, forbidden)
	require.NotErrorIs(t, err, errPrerequisitesNotMet)
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
	assert.Equal(t, status.Error, pm.Status.Phase)

	cond := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.PrerequisitesMet)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Contains(t, cond.Message, forbidden.Error())
}

func TestReconcile_prerequisitesRecover(t *testing.T) {
	req := ctrl.Request{Name: "prometheusmonitoring", Namespace: "dynatrace"}
	pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
	dk := &dynakube.DynaKube{Name: "dk", Namespace: req.Namespace, Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
	secret := &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: map[string][]byte{token.APIKey: []byte("api-token")}}

	gm := newMockGatewayReconciler(t)
	gm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
	tm := newMockTargetAllocatorReconciler(t)
	tm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
	sm := newMockScraperReconciler(t)
	sm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()

	c := fake.NewClient(pm, dk, secret)
	r := NewReconciler(c)
	r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
		return &dynatrace.Client{}, nil
	}
	r.gateway = gm
	r.targetAllocator = tm
	r.scraper = sm

	result, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, missingDependencyRequeueInterval, result.RequeueAfter)

	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(secret), secret))
	secret.Data[token.DataIngestKey] = []byte("data-ingest-token")
	require.NoError(t, c.Update(t.Context(), secret))

	result, err = r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	require.Empty(t, result)

	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
	cond := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.PrerequisitesMet)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, status.Running, pm.Status.Phase)
}

func TestReconcile_dynaKubePhaseIsIgnored(t *testing.T) {
	req := ctrl.Request{Name: "prometheusmonitoring", Namespace: "dynatrace"}

	for _, phase := range []status.DeploymentPhase{status.Deploying, status.Error} {
		t.Run(string(phase), func(t *testing.T) {
			pm := &prometheusmonitoring.PrometheusMonitoring{Name: req.Name, Namespace: req.Namespace, Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: "dk"}}
			dk := &dynakube.DynaKube{
				Name: "dk", Namespace: req.Namespace,
				Spec:   dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"},
				Status: dynakube.DynaKubeStatus{Phase: phase},
			}
			secret := &corev1.Secret{Name: "dk", Namespace: req.Namespace, Data: map[string][]byte{token.APIKey: []byte("api-token"), token.DataIngestKey: []byte("data-ingest-token")}}

			gm := newMockGatewayReconciler(t)
			gm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
			tm := newMockTargetAllocatorReconciler(t)
			tm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()
			sm := newMockScraperReconciler(t)
			sm.EXPECT().Reconcile(t.Context(), mock.Anything, dk, image.Client(nil)).Return(nil).Once()

			c := fake.NewClient(pm, dk, secret)
			r := NewReconciler(c)
			r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
				return &dynatrace.Client{}, nil
			}
			r.gateway = gm
			r.targetAllocator = tm
			r.scraper = sm

			result, err := r.Reconcile(t.Context(), req)

			require.NoError(t, err)
			require.Empty(t, result)
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pm), pm))
			require.Equal(t, status.Running, pm.Status.Phase)
		})
	}
}

func Test_newDynaKubeChangedPredicate(t *testing.T) {
	newDK := func(mutate func(dk *dynakube.DynaKube)) *dynakube.DynaKube {
		dk := &dynakube.DynaKube{Name: "dk", Namespace: "dynatrace", Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.dynatrace.com/api"}}
		if mutate != nil {
			mutate(dk)
		}

		return dk
	}

	tests := []struct {
		name     string
		mutate   func(dk *dynakube.DynaKube)
		expected bool
	}{
		{"no change", nil, false},
		{"phase only", func(dk *dynakube.DynaKube) { dk.Status.Phase = status.Error }, false},
		{"apiUrl", func(dk *dynakube.DynaKube) { dk.Spec.APIURL = "https://other.dynatrace.com/api" }, true},
		{"kubernetes cluster name", func(dk *dynakube.DynaKube) { dk.Status.KubernetesClusterName = "cluster" }, true},
		{"tokens", func(dk *dynakube.DynaKube) { dk.Spec.Tokens = "other-secret" }, true},
		{"trusted CAs", func(dk *dynakube.DynaKube) { dk.Spec.TrustedCAs = "ca-configmap" }, true},
		{"resource attributes", func(dk *dynakube.DynaKube) { dk.Spec.ResourceAttributes = map[string]string{"a": "b"} }, true},
	}

	p := newDynaKubeChangedPredicate()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := event.TypedUpdateEvent[client.Object]{ObjectOld: newDK(nil), ObjectNew: newDK(tt.mutate)}

			assert.Equal(t, tt.expected, p.Update(e))
		})
	}

	t.Run("update with unexpected object type", func(t *testing.T) {
		e := event.TypedUpdateEvent[client.Object]{ObjectOld: &corev1.Secret{}, ObjectNew: &corev1.Secret{}}

		assert.False(t, p.Update(e))
	})

	t.Run("create is ignored", func(t *testing.T) {
		assert.False(t, p.Create(event.TypedCreateEvent[client.Object]{Object: newDK(nil)}))
	})

	t.Run("delete triggers", func(t *testing.T) {
		assert.True(t, p.Delete(event.TypedDeleteEvent[client.Object]{Object: newDK(nil)}))
	})
}

func Test_setPhase(t *testing.T) {
	conditionTrue := metav1.Condition{Type: "Available", Status: metav1.ConditionTrue, Reason: status.ReasonAvailable}
	conditionReconciling := metav1.Condition{Type: "Available", Status: metav1.ConditionFalse, Reason: status.ReasonReconciling}
	conditionError := metav1.Condition{Type: "Available", Status: metav1.ConditionFalse, Reason: status.ReasonError}
	conditionDynaKubeMissing := metav1.Condition{Type: prometheusmonitoring.DynaKubeAvailable, Status: metav1.ConditionFalse, Reason: status.ReasonNotFound}

	boom := errors.New("boom")

	tests := []struct {
		name          string
		err           error
		conditions    []metav1.Condition
		expectedPhase status.DeploymentPhase
		expectedErr   error
	}{
		{"missing dynakube", nil, []metav1.Condition{conditionDynaKubeMissing}, status.Error, nil},
		{"prerequisites not met", errPrerequisitesNotMet, nil, status.Error, nil},
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
		oldProxy *types.ValueSource
		newProxy *types.ValueSource
		expect   bool
	}{
		{"both nil", nil, nil, false},
		{"old nil", nil, &types.ValueSource{Value: "test"}, true},
		{"new nil", &types.ValueSource{Value: "test"}, nil, true},
		{"both empty", &types.ValueSource{}, &types.ValueSource{}, false},
		{"old empty", &types.ValueSource{}, &types.ValueSource{Value: "test"}, true},
		{"new empty", &types.ValueSource{Value: "test"}, &types.ValueSource{}, true},
		{"value equal", &types.ValueSource{Value: "test"}, &types.ValueSource{Value: "test"}, false},
		{"value diff", &types.ValueSource{Value: "foo"}, &types.ValueSource{Value: "bar"}, true},
		{"valueFrom equal", &types.ValueSource{ValueFrom: "test"}, &types.ValueSource{ValueFrom: "test"}, false},
		{"valueFrom diff", &types.ValueSource{ValueFrom: "foo"}, &types.ValueSource{ValueFrom: "bar"}, true},
		{"field diff", &types.ValueSource{Value: "test"}, &types.ValueSource{ValueFrom: "test"}, true},
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

func Test_dynaKubeProxyChangedWithNoProxyFF(t *testing.T) {
	tests := []struct {
		name       string
		oldProxy   *types.ValueSource
		oldNoProxy string
		newProxy   *types.ValueSource
		newNoProxy string
		expect     bool
	}{
		{"values nil, both no-proxy ffs empty", nil, "", nil, "", false},
		{"values nil, new no-proxy ff empty", nil, "test", nil, "", false},
		{"values nil, old no-proxy ff empty", nil, "", nil, "test", false},
		{"values empty, both no-proxy ffs empty", &types.ValueSource{}, "", &types.ValueSource{}, "", false},
		{"values empty, new no-proxy ff empty", &types.ValueSource{}, "test", &types.ValueSource{}, "", false},
		{"values empty, old no-proxy ff empty", &types.ValueSource{}, "", &types.ValueSource{}, "test", false},
		{"values nil, ffs equal", nil, "test", nil, "test", false},
		{"values empty, ffs equal", &types.ValueSource{}, "test", &types.ValueSource{}, "test", false},
		{"values equal, ffs equal", &types.ValueSource{Value: "foo"}, "test", &types.ValueSource{Value: "foo"}, "test", false},
		{"values equal, old no-proxy ff empty", &types.ValueSource{Value: "foo"}, "", &types.ValueSource{Value: "foo"}, "test", true},
		{"values equal, new no-proxy ff empty", &types.ValueSource{Value: "foo"}, "test", &types.ValueSource{Value: "foo"}, "", true},
		{"values diff, no-proxy ff equal", &types.ValueSource{Value: "foo"}, "test", &types.ValueSource{Value: "bar"}, "test", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dynaKubeProxyChanged(
				&dynakube.DynaKube{Annotations: map[string]string{exp.NoProxyKey: tt.oldNoProxy}, Spec: dynakube.DynaKubeSpec{Proxy: tt.oldProxy}},
				&dynakube.DynaKube{Annotations: map[string]string{exp.NoProxyKey: tt.newNoProxy}, Spec: dynakube.DynaKubeSpec{Proxy: tt.newProxy}},
			)
			assert.Equal(t, tt.expect, got)
		})
	}
}

