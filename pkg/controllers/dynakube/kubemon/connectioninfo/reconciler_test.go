package connectioninfo_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

		dtClient := agclientmock.NewClient(t)
		dtClient.EXPECT().GetConnectionInfo(anyCtx).Return(agclient.ConnectionInfo{}, errors.New("dt api error")).Once()

		err := r.Reconcile(t.Context(), dtClient, dk)
		require.Error(t, err)

		assert.Empty(t, dk.Status.KubernetesMonitoring.ConnectionInfo)

		assertResources(t, fakeClient, dk, false, false)
	})

	t.Run("returns transient error when connection info is incomplete, creates no resources", func(t *testing.T) {
		tests := map[string]func(*agclient.ConnectionInfo){
			"empty tenant UUID":  func(info *agclient.ConnectionInfo) { info.TenantUUID = "" },
			"empty endpoints":    func(info *agclient.ConnectionInfo) { info.Endpoints = "" },
			"empty tenant token": func(info *agclient.ConnectionInfo) { info.TenantToken = "" },
		}

		for name, mutate := range tests {
			t.Run(name, func(t *testing.T) {
				dk := newTestDynaKube(true)
				fakeClient := fake.NewClient(dk)
				r := connectioninfo.NewReconciler(fakeClient)

				info := testConnectionInfo()
				mutate(&info)
				dtClient := newDTClientMock(t, info)

				err := r.Reconcile(t.Context(), dtClient, dk)
				require.Error(t, err)

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
	tests := map[string]struct {
		failOn          func(client.Object) bool
		configMapExists bool
		secretExists    bool
	}{
		"configmap write fails": {
			failOn:          isType[*corev1.ConfigMap],
			configMapExists: false,
			secretExists:    false,
		},
		"secret write fails": {
			failOn:          isType[*corev1.Secret],
			configMapExists: true,
			secretExists:    false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if test.failOn(obj) {
						return errors.New("kube api error")
					}

					return c.Create(ctx, obj, opts...)
				},
			}, dk)
			r := connectioninfo.NewReconciler(fakeClient)
			dtClient := newDTClientMock(t, testConnectionInfo())

			err := r.Reconcile(t.Context(), dtClient, dk)
			require.Error(t, err)

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

	tests := map[string]func(client.Object) bool{
		"configmap update fails": isType[*corev1.ConfigMap],
		"secret update fails":    isType[*corev1.Secret],
	}

	for name, failOn := range tests {
		t.Run(name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if failOn(obj) {
						return errors.New("kube api error")
					}

					return c.Update(ctx, obj, opts...)
				},
			}, seed(dk)...)
			r := connectioninfo.NewReconciler(fakeClient)
			dtClient := newDTClientMock(t, testConnectionInfo())

			err := r.Reconcile(t.Context(), dtClient, dk)
			require.Error(t, err)

			assert.Equal(t, oldUUID, dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID)
			assert.Equal(t, oldEndpoints, dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints)
		})
	}
}

// TestReconcileCleanupDeleteFailures covers delete failures per resource on the cleanup path.
func TestReconcileCleanupDeleteFailures(t *testing.T) {
	tests := map[string]func(client.Object) bool{
		"configmap delete fails": isType[*corev1.ConfigMap],
		"secret delete fails":    isType[*corev1.Secret],
	}

	for name, failOn := range tests {
		t.Run(name, func(t *testing.T) {
			dk := newTestDynaKube(false)
			fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if failOn(obj) {
						return errors.New("kube api error")
					}

					return c.Delete(ctx, obj, opts...)
				},
			}, dk)
			r := connectioninfo.NewReconciler(fakeClient)

			err := r.Reconcile(t.Context(), nil, dk)
			require.Error(t, err)
		})
	}
}

// TestReconcileCleanup covers cleanup success across all resource subsets. Delete is IgnoreNotFound,
// so cleanup must succeed regardless of which resources exist and always leave status empty.
func TestReconcileCleanup(t *testing.T) {
	// Delete is IgnoreNotFound, so cleanup must succeed regardless of which subset of
	// resources exists and must always leave neither object and an empty status.
	tests := map[string]struct {
		seedConfigMap bool
		seedSecret    bool
	}{
		"both present":           {seedConfigMap: true, seedSecret: true},
		"only configmap present": {seedConfigMap: true, seedSecret: false},
		"only secret present":    {seedConfigMap: false, seedSecret: true},
		"nothing present":        {seedConfigMap: false, seedSecret: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
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

	err := c.Get(t.Context(), types.NamespacedName{Name: name, Namespace: testNamespace}, into)
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
