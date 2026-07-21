// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package connectioninfo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// Unit tests for the connectioninfo reconciler. Use a fake client with interceptors to inject
// write/delete failures and a mocked Dynatrace client for the API call; they own all branch and
// error logic. The multi-reconcile lifecycle is covered by the integration test.

const (
	testName        = "test-dk"
	testNamespace   = "dynatrace"
	testTenantUUID  = "test-uuid"
	testEndpoints   = "https://tenant.live.dynatrace.com/communication"
	testTenantToken = "test-token"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

// TestReconcilePreconditionErrors covers input errors that abort before any write: a failing
// Dynatrace API call and each missing field in the returned connection info.
func TestReconcilePreconditionErrors(t *testing.T) {
	t.Run("returns error when getting connection info fails", func(t *testing.T) {
		dk := newTestDynaKube(true)
		fakeClient := fake.NewClient(dk)
		r := connectioninfo.NewReconciler(fakeClient)

		dtErr := errors.New("dt api error")
		dtClient := agclientmock.NewClient(t)
		dtClient.EXPECT().GetConnectionInfo(anyCtx).Return(agclient.ConnectionInfo{}, dtErr).Once()

		err := r.Reconcile(t.Context(), dtClient, dk)
		require.ErrorIs(t, err, dtErr)

		assert.Empty(t, dk.Status.KubernetesMonitoring.ConnectionInfo)

		assertResources(t, fakeClient, dk, false, false)
	})

	t.Run("returns transient error when connection info is incomplete, creates no resources", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*agclient.ConnectionInfo)
		}{
			{"empty tenant UUID", func(info *agclient.ConnectionInfo) { info.TenantUUID = "" }},
			{"empty endpoints", func(info *agclient.ConnectionInfo) { info.Endpoints = "" }},
			{"empty tenant token", func(info *agclient.ConnectionInfo) { info.TenantToken = "" }},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				dk := newTestDynaKube(true)
				fakeClient := fake.NewClient(dk)
				r := connectioninfo.NewReconciler(fakeClient)

				info := testConnectionInfo()
				test.mutate(&info)
				dtClient := newDTClientMock(t, info)

				err := r.Reconcile(t.Context(), dtClient, dk)
				require.ErrorIs(t, err, connectioninfo.ErrConnectionInfoNotReady)

				assertResources(t, fakeClient, dk, false, false)
			})
		}
	})
}

// TestReconcileWriteFailures covers create-path failures per resource. A ConfigMap failure aborts
// before any write; a Secret failure leaves the ConfigMap behind — both must leave status empty.
func TestReconcileWriteFailures(t *testing.T) {
	// In both cases status must stay empty and the failing write is never persisted. They differ
	// only in whether the ConfigMap survives: a ConfigMap failure aborts before either object is
	// written, while a Secret failure happens after the ConfigMap was already created.
	tests := []struct {
		name            string
		failOn          func(client.Object) bool
		configMapExists bool
		secretExists    bool
	}{
		{"configmap write fails", isType[*corev1.ConfigMap], false, false},
		{"secret write fails", isType[*corev1.Secret], true, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			errCreate := errors.New("kube api error")
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if test.failOn(obj) {
						return errCreate
					}

					return c.Create(ctx, obj, opts...)
				},
			}, dk)
			r := connectioninfo.NewReconciler(fakeClient)
			dtClient := newDTClientMock(t, testConnectionInfo())

			err := r.Reconcile(t.Context(), dtClient, dk)
			require.ErrorIs(t, err, errCreate)

			assert.Empty(t, dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID)
			assert.Empty(t, dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints)

			assertResources(t, fakeClient, dk, test.configMapExists, test.secretExists)
		})
	}
}

// TestReconcileRotationFailures covers update-path failures when resources already exist. Pre-seeds
// both resources and a prior status to assert that status is not advanced on a failed rotation.
func TestReconcileRotationFailures(t *testing.T) {
	const (
		oldUUID      = "old-uuid"
		oldEndpoints = "https://old.live.dynatrace.com/communication"
	)

	// seed pre-existing resources and an already-populated status so CreateOrUpdate
	// takes the update path and a failed rotation can be observed against prior values.
	seed := func(dk *dynakube.DynaKube) []client.Object {
		dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID = oldUUID
		dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints = oldEndpoints

		return []client.Object{
			dk,
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), Namespace: testNamespace}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: testNamespace}},
		}
	}

	tests := []struct {
		name   string
		failOn func(client.Object) bool
	}{
		{"configmap update fails", isType[*corev1.ConfigMap]},
		{"secret update fails", isType[*corev1.Secret]},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			errUpdate := errors.New("kube api error")
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if test.failOn(obj) {
						return errUpdate
					}

					return c.Update(ctx, obj, opts...)
				},
			}, seed(dk)...)
			r := connectioninfo.NewReconciler(fakeClient)
			dtClient := newDTClientMock(t, testConnectionInfo())

			err := r.Reconcile(t.Context(), dtClient, dk)
			require.ErrorIs(t, err, errUpdate)

			assert.Equal(t, oldUUID, dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID)
			assert.Equal(t, oldEndpoints, dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints)
		})
	}
}

// TestReconcileCleanupDeleteFailures covers delete failures per resource on the cleanup path.
func TestReconcileCleanupDeleteFailures(t *testing.T) {
	tests := []struct {
		name   string
		failOn func(client.Object) bool
	}{
		{"configmap delete fails", isType[*corev1.ConfigMap]},
		{"secret delete fails", isType[*corev1.Secret]},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dk := newTestDynaKube(false)
			errDelete := errors.New("kube api error")
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if test.failOn(obj) {
						return errDelete
					}

					return c.Delete(ctx, obj, opts...)
				},
			}, dk)
			r := connectioninfo.NewReconciler(fakeClient)

			err := r.Reconcile(t.Context(), nil, dk)
			require.ErrorIs(t, err, errDelete)
		})
	}
}

// TestReconcileCleanup covers cleanup success across all resource subsets. Delete is IgnoreNotFound,
// so cleanup must succeed regardless of which resources exist and always leave status empty.
func TestReconcileCleanup(t *testing.T) {
	// Delete is IgnoreNotFound, so cleanup must succeed regardless of which subset of
	// resources exists and must always leave neither object and an empty status.
	tests := []struct {
		name          string
		seedConfigMap bool
		seedSecret    bool
	}{
		{"both present", true, true},
		{"only configmap present", true, false},
		{"only secret present", false, true},
		{"nothing present", false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dk := newTestDynaKube(false)
			dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID = testTenantUUID
			dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints = testEndpoints

			objs := []client.Object{dk}
			if test.seedConfigMap {
				objs = append(objs, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), Namespace: testNamespace}})
			}

			if test.seedSecret {
				objs = append(objs, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: testNamespace}})
			}

			fakeClient := fake.NewClient(objs...)
			r := connectioninfo.NewReconciler(fakeClient)

			err := r.Reconcile(t.Context(), nil, dk)
			require.NoError(t, err)

			assertResources(t, fakeClient, dk, false, false)
			assert.Empty(t, dk.Status.KubernetesMonitoring.ConnectionInfo)
		})
	}
}

func newTestDynaKube(enabled bool) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
		},
	}

	if enabled {
		dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{}
	}

	return dk
}

func testConnectionInfo() agclient.ConnectionInfo {
	return agclient.ConnectionInfo{
		TenantUUID:  testTenantUUID,
		TenantToken: testTenantToken,
		Endpoints:   testEndpoints,
	}
}

func newDTClientMock(t *testing.T, info agclient.ConnectionInfo) *agclientmock.Client {
	t.Helper()
	m := agclientmock.NewClient(t)
	m.EXPECT().GetConnectionInfo(anyCtx).Return(info, nil).Once()

	return m
}

func isType[T client.Object](obj client.Object) bool {
	_, ok := obj.(T)

	return ok
}

func assertExists(t *testing.T, c client.Client, into client.Object, name string, wantExists bool) {
	t.Helper()

	err := c.Get(t.Context(), client.ObjectKey{Name: name, Namespace: testNamespace}, into)
	if wantExists {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
	}
}

func assertResources(t *testing.T, c client.Client, dk *dynakube.DynaKube, configMapExists, secretExists bool) {
	t.Helper()

	assertExists(t, c, &corev1.ConfigMap{}, dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), configMapExists)
	assertExists(t, c, &corev1.Secret{}, dk.KubernetesMonitoring().GetTenantSecretName(), secretExists)
}
