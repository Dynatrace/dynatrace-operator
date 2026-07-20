package authtoken_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace    = "dynatrace"
	testDynakubeName = "test-dk"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func newTestDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
			KubernetesMonitoring: &kubemonapi.Spec{
				StatefulSetProperties: kubemonapi.StatefulSetProperties{
					Image: "registry.example.com/linux/activegate:1.2.3",
				},
			},
		},
	}
}

func newFreshSecret(dk *dynakube.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
			Namespace: dk.Namespace,
			// CreationTimestamp must be set explicitly: the fake client stores whatever is
			// in the object (it has no server-side clock), so a zero timestamp would always
			// appear outdated. Real k8s API server sets this on Create and ignores it on
			// updates, so this value is meaningless in production.
			CreationTimestamp: metav1.Now(),
		},
		Data: map[string][]byte{authtoken.SecretKey: []byte(token)},
	}
}

func getAuthTokenSecret(t *testing.T, clt client.Client, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{
		Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
		Namespace: dk.Namespace,
	}, secret))

	return secret
}

func TestReconcile(t *testing.T) {
	t.Run("creates secret on first reconcile", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)
		agCl := agclientmock.NewClient(t)
		agCl.EXPECT().GetAuthToken(anyCtx, dk.Name).Return(newAuthTokenResponse("first-token"), nil).Once()

		r := authtoken.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), agCl, dk))

		secret := getAuthTokenSecret(t, clt, dk)
		assert.Equal(t, []byte("first-token"), secret.Data[authtoken.SecretKey])
	})

	t.Run("no-op when secret is fresh", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk, newFreshSecret(dk, "existing-token"))
		agCl := agclientmock.NewClient(t)
		// No GetAuthToken expectation — the mock will fail if it is called.

		r := authtoken.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), agCl, dk))
	})

	t.Run("rotates outdated secret — creation timestamp resets so next reconcile is no-op", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			dk := newTestDynaKube()
			fresh := newFreshSecret(dk, "old-token")
			clt := fake.NewClient(dk, fresh)
			agCl := agclientmock.NewClient(t)
			agCl.EXPECT().GetAuthToken(anyCtx, dk.Name).Return(newAuthTokenResponse("new-token"), nil).Once()

			r := authtoken.NewReconciler(clt)

			// Fast-forwards the bubble's fake clock past the real rotation interval instead of
			// waiting it out or backdating CreationTimestamp by hand.
			time.Sleep(authtoken.DefaultRotationInterval + time.Second)

			require.NoError(t, r.Reconcile(t.Context(), agCl, dk))

			rotated := getAuthTokenSecret(t, clt, dk)
			assert.Equal(t, []byte("new-token"), rotated.Data[authtoken.SecretKey])

			// Simulate what the real API server does on Create: it sets CreationTimestamp to
			// the current time. The fake client does not do this, so we set it manually here
			// to reflect the state the production reconciler would observe after rotation.
			rotated.CreationTimestamp = metav1.Now()
			require.NoError(t, clt.Update(t.Context(), rotated))

			// Second reconcile must be a no-op: mock has no remaining GetAuthToken expectations
			// and will panic if called, proving rotation does not loop.
			require.NoError(t, r.Reconcile(t.Context(), agCl, dk))
		})
	})

	t.Run("cleans up secret when kubemon disabled", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk, newFreshSecret(dk, "some-token"))
		agCl := agclientmock.NewClient(t)

		r := authtoken.NewReconciler(clt)

		dk.Spec.KubernetesMonitoring = nil

		require.NoError(t, r.Reconcile(t.Context(), agCl, dk))

		err := clt.Get(t.Context(), types.NamespacedName{
			Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
			Namespace: dk.Namespace,
		}, &corev1.Secret{})
		assert.True(t, k8serrors.IsNotFound(err), "secret should be deleted when kubemon is disabled")
	})
}

// TestReconcilePreconditionErrors covers the Dynatrace API call failing before any write.
func TestReconcilePreconditionErrors(t *testing.T) {
	t.Run("returns error when getting auth token fails, creates no secret", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)
		agCl := agclientmock.NewClient(t)
		agCl.EXPECT().GetAuthToken(anyCtx, dk.Name).Return(nil, errors.New("dt api error")).Once()

		r := authtoken.NewReconciler(clt)

		require.Error(t, r.Reconcile(t.Context(), agCl, dk))

		err := clt.Get(t.Context(), types.NamespacedName{
			Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
			Namespace: dk.Namespace,
		}, &corev1.Secret{})
		assert.True(t, k8serrors.IsNotFound(err), "secret must not be created when the Dynatrace API call fails")
	})
}

// TestReconcileWriteFailures covers Kubernetes API failures on the read/create path.
func TestReconcileWriteFailures(t *testing.T) {
	t.Run("returns error when getting the secret fails with a non-NotFound error", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errors.New("kube api error")
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk)
		agCl := agclientmock.NewClient(t)
		// No GetAuthToken expectation — a failed Get must abort before the Dynatrace API is called.

		r := authtoken.NewReconciler(clt)

		require.Error(t, r.Reconcile(t.Context(), agCl, dk))
	})

	t.Run("returns error when secret creation fails", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				return errors.New("kube api error")
			},
		}, dk)
		agCl := agclientmock.NewClient(t)
		agCl.EXPECT().GetAuthToken(anyCtx, dk.Name).Return(newAuthTokenResponse("first-token"), nil).Once()

		r := authtoken.NewReconciler(clt)

		require.Error(t, r.Reconcile(t.Context(), agCl, dk))
	})
}

// TestReconcileRotationFailures covers a failing delete on the rotation path: the outdated secret
// must survive and no new token must be fetched, since rotation deletes before creating.
func TestReconcileRotationFailures(t *testing.T) {
	t.Run("returns error and leaves outdated secret in place when delete fails", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			dk := newTestDynaKube()
			fresh := newFreshSecret(dk, "old-token")
			clt := fake.NewClientWithInterceptors(interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					return errors.New("kube api error")
				},
			}, dk, fresh)
			agCl := agclientmock.NewClient(t)
			// No GetAuthToken expectation — a failed delete must abort before a new token is fetched.

			r := authtoken.NewReconciler(clt)

			time.Sleep(authtoken.DefaultRotationInterval + time.Second)

			require.Error(t, r.Reconcile(t.Context(), agCl, dk))

			secret := getAuthTokenSecret(t, clt, dk)
			assert.Equal(t, []byte("old-token"), secret.Data[authtoken.SecretKey])
		})
	})
}

// TestReconcileCleanupFailures covers a failing delete on the cleanup-on-disable path.
func TestReconcileCleanupFailures(t *testing.T) {
	t.Run("returns error when secret deletion fails", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return errors.New("kube api error")
			},
		}, dk, newFreshSecret(dk, "some-token"))
		agCl := agclientmock.NewClient(t)
		dk.Spec.KubernetesMonitoring = nil

		r := authtoken.NewReconciler(clt)

		require.Error(t, r.Reconcile(t.Context(), agCl, dk))
	})
}

func newAuthTokenResponse(token string) *agclient.AuthTokenInfo {
	return &agclient.AuthTokenInfo{
		TokenID: "test-id",
		Token:   token,
	}
}
