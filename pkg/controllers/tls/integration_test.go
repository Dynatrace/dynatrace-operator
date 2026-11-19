// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	operatortls "github.com/Dynatrace/dynatrace-operator/pkg/controllers/tls"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	integrationNamespace = "dynatrace"
	eventuallyTimeout    = 15 * time.Second
	eventuallyInterval   = 100 * time.Millisecond
	neverTimeout         = 2 * time.Second
)

func caSecretKey() types.NamespacedName {
	return types.NamespacedName{Name: operatortls.CASecretName, Namespace: integrationNamespace}
}

func bundleKey() types.NamespacedName {
	return types.NamespacedName{Name: operatortls.CABundleConfigMapName, Namespace: integrationNamespace}
}

func getCASecret(t *testing.T, clt client.Client) (*corev1.Secret, error) {
	t.Helper()

	secret := &corev1.Secret{}
	err := clt.Get(t.Context(), caSecretKey(), secret)

	return secret, err
}

func getBundle(t *testing.T, clt client.Client) (*corev1.ConfigMap, error) {
	t.Helper()

	cm := &corev1.ConfigMap{}
	err := clt.Get(t.Context(), bundleKey(), cm)

	return cm, err
}

// caChanged reports whether the CA objects were removed or modified compared to the given state.
func caChanged(t *testing.T, clt client.Client, secretBefore *corev1.Secret, cmBefore *corev1.ConfigMap) bool {
	t.Helper()

	secret, err := getCASecret(t, clt)
	if err != nil {
		return true
	}

	cm, err := getBundle(t, clt)
	if err != nil {
		return true
	}

	return secret.ResourceVersion != secretBefore.ResourceVersion || cm.ResourceVersion != cmBefore.ResourceVersion
}

func requireCondition(t *testing.T, dk *dynakube.DynaKube, status metav1.ConditionStatus, reason string) {
	t.Helper()

	cond := meta.FindStatusCondition(dk.Status.Conditions, operatortls.ConditionType)
	require.NotNil(t, cond)
	assert.Equal(t, status, cond.Status)
	assert.Equal(t, reason, cond.Reason)
}

// requireCAReplaced waits until the CA differs from old, is free of the corrupted key and the bundle matches it.
func requireCAReplaced(t *testing.T, clt client.Client, old *corev1.Secret) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		current, err := getCASecret(t, clt)
		if !assert.NoError(c, err) {
			return
		}

		assert.NotEqual(c, old.Data[operatortls.CACertKey], current.Data[operatortls.CACertKey])
		assert.NotEqual(c, old.Data[operatortls.CAKeyKey], current.Data[operatortls.CAKeyKey])

		cm, err := getBundle(t, clt)
		if !assert.NoError(c, err) {
			return
		}

		assert.Equal(c, string(current.Data[operatortls.CACertKey]), cm.Data[operatortls.CACertKey])
		assert.NotContains(c, cm.Data, operatortls.CAKeyKey)
	}, eventuallyTimeout, eventuallyInterval)
}

func TestOperatorRootCA(t *testing.T) {
	clt := integrationtests.SetupManagerTestEnvironment(t, func(mgr ctrl.Manager) error {
		return operatortls.Add(mgr, integrationNamespace)
	})

	integrationtests.CreateNamespace(t, clt, integrationNamespace)

	// DynaKubes are created with the parent test, so that they outlive the subtest that creates them
	parent := t

	dk := &dynakube.DynaKube{
		Name:      "first",
		Namespace: integrationNamespace,
		Spec:      dynakube.DynaKubeSpec{APIURL: "https://tenant.live.dynatrace.com/api"},
	}

	t.Run("operator start-up creates the CA without any DynaKube", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			secret, err := getCASecret(t, clt)
			if !assert.NoError(c, err) {
				return
			}

			assert.Empty(c, secret.OwnerReferences)
			assert.Contains(c, secret.Data, operatortls.CACertKey)
			assert.Contains(c, secret.Data, operatortls.CAKeyKey)

			cm, err := getBundle(t, clt)
			if !assert.NoError(c, err) {
				return
			}

			assert.Empty(c, cm.OwnerReferences)
			assert.NotContains(c, cm.Data, operatortls.CAKeyKey)
			assert.Equal(c, string(secret.Data[operatortls.CACertKey]), cm.Data[operatortls.CACertKey])
		}, eventuallyTimeout, eventuallyInterval)
	})

	t.Run("reconciling a DynaKube does not change the CA", func(t *testing.T) {
		integrationtests.CreateDynakube(parent, clt, dk)

		before, err := getCASecret(t, clt)
		require.NoError(t, err)

		require.NoError(t, operatortls.VerifyCA(t.Context(), clt, dk))
		require.NoError(t, operatortls.VerifyCA(t.Context(), clt, dk))

		requireCondition(t, dk, metav1.ConditionTrue, operatortls.ReasonCAReady)

		after, err := getCASecret(t, clt)
		require.NoError(t, err)
		assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
	})

	t.Run("a second DynaKube with a different spec shares the CA", func(t *testing.T) {
		// TODO: also cover explicit `.spec.tls.mode: Off` once ICP-11033 lands, the CA must exist in every mode.
		dk2 := &dynakube.DynaKube{
			Name:      "second",
			Namespace: integrationNamespace,
			Spec:      dynakube.DynaKubeSpec{APIURL: "https://other.live.dynatrace.com/api", EnableIstio: new(false)},
		}
		integrationtests.CreateDynakube(parent, clt, dk2)

		before, err := getCASecret(t, clt)
		require.NoError(t, err)

		require.NoError(t, operatortls.VerifyCA(t.Context(), clt, dk2))
		requireCondition(t, dk2, metav1.ConditionTrue, operatortls.ReasonCAReady)

		after, err := getCASecret(t, clt)
		require.NoError(t, err)
		assert.Equal(t, before.UID, after.UID)
		assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
	})

	t.Run("DynaKube deletion keeps the CA", func(t *testing.T) {
		secretBefore, err := getCASecret(t, clt)
		require.NoError(t, err)

		cmBefore, err := getBundle(t, clt)
		require.NoError(t, err)

		require.NoError(t, clt.Delete(t.Context(), dk))

		assert.Never(t, func() bool { return caChanged(t, clt, secretBefore, cmBefore) }, neverTimeout, eventuallyInterval)

		secretAfter, err := getCASecret(t, clt)
		require.NoError(t, err)
		assert.Empty(t, secretAfter.OwnerReferences)
	})

	t.Run("corrupted key is replaced by a new valid CA", func(t *testing.T) {
		secret, err := getCASecret(t, clt)
		require.NoError(t, err)

		corrupted := corruptCAKey(t, clt, secret)

		requireCAReplaced(t, clt, corrupted)

		dk3 := &dynakube.DynaKube{Name: "third", Namespace: integrationNamespace}
		require.NoError(t, operatortls.VerifyCA(t.Context(), clt, dk3))
		requireCondition(t, dk3, metav1.ConditionTrue, operatortls.ReasonCAReady)
	})

	t.Run("deleting the secret creates a new CA and updates the bundle", func(t *testing.T) {
		old, err := getCASecret(t, clt)
		require.NoError(t, err)

		require.NoError(t, clt.Delete(t.Context(), old))

		requireCAReplaced(t, clt, old)
	})
}

// corruptCAKey replaces the (immutable) CA Secret with one that has a broken key. The tls controller may recreate the Secret
// in between, so it is retried until the corrupted Secret was created.
func corruptCAKey(t *testing.T, clt client.Client, valid *corev1.Secret) *corev1.Secret {
	t.Helper()

	corrupted := &corev1.Secret{
		Name: valid.Name, Namespace: valid.Namespace, Labels: valid.Labels,
		Immutable: valid.Immutable,
		Data: map[string][]byte{
			operatortls.CACertKey: valid.Data[operatortls.CACertKey],
			operatortls.CAKeyKey:  []byte("garbage"),
		},
	}

	var created *corev1.Secret

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		current := &corev1.Secret{}
		if err := clt.Get(t.Context(), caSecretKey(), current); err == nil {
			require.NoError(c, clt.Delete(t.Context(), current))
		}

		attempt := corrupted.DeepCopy()
		if assert.NoError(c, clt.Create(t.Context(), attempt)) {
			created = attempt
		}
	}, eventuallyTimeout, eventuallyInterval)

	return created
}

func TestVerifyCANeverCreatesCA(t *testing.T) {
	// no tls controller is running in this environment
	clt := integrationtests.SetupManagerTestEnvironment(t, func(ctrl.Manager) error { return nil })

	integrationtests.CreateNamespace(t, clt, integrationNamespace)

	dk := &dynakube.DynaKube{
		Name:      "alone",
		Namespace: integrationNamespace,
	}
	require.NoError(t, operatortls.VerifyCA(t.Context(), clt, dk))
	requireCondition(t, dk, metav1.ConditionFalse, operatortls.ReasonCANotFound)

	_, err := getCASecret(t, clt)
	require.Error(t, err)

	_, err = getBundle(t, clt)
	require.Error(t, err)
}

func TestWatchCA(t *testing.T) {
	var (
		mu         sync.Mutex
		reconciled = map[string]int{}
	)

	reset := func() {
		mu.Lock()
		defer mu.Unlock()

		reconciled = map[string]int{}
	}

	count := func(name string) int {
		mu.Lock()
		defer mu.Unlock()

		return reconciled[name]
	}

	// controller names have to be unique in a process, so the tls controller is not started here,
	// the CA is created by calling its Reconcile directly
	clt := integrationtests.SetupManagerTestEnvironment(t, func(mgr ctrl.Manager) error {
		consumer := reconcile.Func(func(_ context.Context, req reconcile.Request) (reconcile.Result, error) {
			mu.Lock()
			defer mu.Unlock()

			reconciled[req.Name]++

			return reconcile.Result{}, nil
		})

		// the consumer decides what to reconcile, here all DynaKubes
		mapFn := func(ctx context.Context, _ client.Object) []reconcile.Request {
			dkList := &dynakube.DynaKubeList{}
			if err := mgr.GetClient().List(ctx, dkList); err != nil {
				return nil
			}

			requests := make([]reconcile.Request, 0, len(dkList.Items))
			for _, dk := range dkList.Items {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&dk)})
			}

			return requests
		}

		return operatortls.WatchCA(ctrl.NewControllerManagedBy(mgr).Named("watch-ca-test").For(&dynakube.DynaKube{}), integrationNamespace, mapFn).
			Complete(consumer)
	})

	integrationtests.CreateNamespace(t, clt, integrationNamespace)

	for _, name := range []string{"first", "second"} {
		integrationtests.CreateDynakube(t, clt, &dynakube.DynaKube{
			Name:      name,
			Namespace: integrationNamespace,
			Spec:      dynakube.DynaKubeSpec{APIURL: "https://tenant.live.dynatrace.com/api"},
		})
	}

	// let the reconciles caused by the creation of the DynaKubes settle
	settle := func() {
		require.Eventually(t, func() bool {
			before := count("first") + count("second")
			time.Sleep(500 * time.Millisecond)

			return count("first")+count("second") == before
		}, eventuallyTimeout, eventuallyInterval)

		reset()
	}

	requireAllReconciled := func(t *testing.T, trigger func()) {
		t.Helper()

		settle()
		trigger()

		assert.Eventually(t, func() bool { return count("first") > 0 && count("second") > 0 }, eventuallyTimeout, eventuallyInterval)
	}

	t.Run("creation of the CA reconciles all DynaKubes", func(t *testing.T) {
		requireAllReconciled(t, func() {
			_, err := operatortls.NewController(clt, integrationNamespace).Reconcile(t.Context(), reconcile.Request{})
			require.NoError(t, err)
		})
	})

	t.Run("change of the bundle reconciles all DynaKubes", func(t *testing.T) {
		requireAllReconciled(t, func() {
			bundle, err := getBundle(t, clt)
			require.NoError(t, err)

			bundle.Data["unrelated"] = "change"
			require.NoError(t, clt.Update(t.Context(), bundle))
		})
	})

	t.Run("deletion of the CA reconciles all DynaKubes", func(t *testing.T) {
		requireAllReconciled(t, func() {
			secret, err := getCASecret(t, clt)
			require.NoError(t, err)

			require.NoError(t, clt.Delete(t.Context(), secret))
		})
	})

	t.Run("unrelated objects do not reconcile the DynaKubes", func(t *testing.T) {
		settle()

		require.NoError(t, clt.Create(t.Context(), &corev1.Secret{Name: "unrelated", Namespace: integrationNamespace}))
		require.NoError(t, clt.Create(t.Context(), &corev1.ConfigMap{Name: "unrelated", Namespace: integrationNamespace}))

		assert.Never(t, func() bool { return count("first") > 0 || count("second") > 0 }, neverTimeout, eventuallyInterval)
	})
}
