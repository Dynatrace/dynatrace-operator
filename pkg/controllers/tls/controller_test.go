// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testNamespace = "dynatrace"

func reconcileOnce(t *testing.T, c *Controller) {
	t.Helper()

	result, err := c.Reconcile(t.Context(), reconcile.Request{})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func getSecret(t *testing.T, clt client.Client) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: CASecretName, Namespace: testNamespace}, secret))

	return secret
}

func getConfigMap(t *testing.T, clt client.Client) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: CABundleConfigMapName, Namespace: testNamespace}, cm))

	return cm
}

func TestLoadCA(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		clt := fake.NewClient()

		_, err := loadCA(t.Context(), clt, testNamespace, time.Now())
		require.ErrorIs(t, err, ErrCANotFound)
		assert.False(t, k8serrors.IsNotFound(err))
	})

	t.Run("invalid", func(t *testing.T) {
		clt := fake.NewClient()
		_, err := NewController(clt, testNamespace).Reconcile(t.Context(), reconcile.Request{})
		require.NoError(t, err)

		secret := getSecret(t, clt)
		secret.Data[CAKeyKey] = []byte("garbage")
		require.NoError(t, clt.Update(t.Context(), secret))

		_, err = loadCA(t.Context(), clt, testNamespace, time.Now())
		require.ErrorIs(t, err, certificates.ErrInvalidCA)
	})
}

func TestReconcile(t *testing.T) {
	t.Run("creates secret and bundle on empty cluster", func(t *testing.T) {
		clt := fake.NewClient()

		reconcileOnce(t, NewController(clt, testNamespace))

		secret := getSecret(t, clt)
		assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
		require.NotNil(t, secret.Immutable)
		assert.True(t, *secret.Immutable)
		assert.Len(t, secret.Data, 2)
		assert.Contains(t, secret.Data, CACertKey)
		assert.Contains(t, secret.Data, CAKeyKey)
		assert.Equal(t, k8slabel.OperatorComponentLabel, secret.Labels[k8slabel.AppNameLabel])
		assert.Empty(t, secret.OwnerReferences, "there is no operator deployment to own the secret")

		cm := getConfigMap(t, clt)
		assert.Len(t, cm.Data, 1)
		assert.Equal(t, string(secret.Data[CACertKey]), cm.Data[CACertKey])
		assert.Equal(t, k8slabel.OperatorComponentLabel, cm.Labels[k8slabel.AppNameLabel])
		require.NotNil(t, cm.Immutable)
		assert.True(t, *cm.Immutable)
		require.Len(t, cm.OwnerReferences, 1)
		assert.Equal(t, "Secret", cm.OwnerReferences[0].Kind)
		assert.Equal(t, CASecretName, cm.OwnerReferences[0].Name)
		assert.Nil(t, cm.OwnerReferences[0].BlockOwnerDeletion)
	})

	t.Run("secret is owned by the operator deployment", func(t *testing.T) {
		deployment := &appsv1.Deployment{Name: version.AppName, Namespace: testNamespace, UID: "deployment-uid"}
		clt := fake.NewClient(deployment)
		c := NewController(clt, testNamespace)

		reconcileOnce(t, c)

		secret := getSecret(t, clt)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, "Deployment", secret.OwnerReferences[0].Kind)
		assert.Equal(t, version.AppName, secret.OwnerReferences[0].Name)
		assert.True(t, metav1.IsControlledBy(secret, deployment))

		// owning it does not make the secret change on every reconcile
		rv := secret.ResourceVersion

		reconcileOnce(t, c)

		assert.Equal(t, rv, getSecret(t, clt).ResourceVersion)
	})

	t.Run("bundle is recreated if the CA is regenerated", func(t *testing.T) {
		clt := fake.NewClient()
		c := NewController(clt, testNamespace)

		reconcileOnce(t, c)

		oldCert := getSecret(t, clt).Data[CACertKey]

		require.NoError(t, clt.Delete(t.Context(), getSecret(t, clt)))
		reconcileOnce(t, c)

		newCert := getSecret(t, clt).Data[CACertKey]
		require.NotEqual(t, oldCert, newCert)

		cm := getConfigMap(t, clt)
		assert.Equal(t, string(newCert), cm.Data[CACertKey])
		assert.True(t, *cm.Immutable)
	})

	t.Run("second reconcile does not change anything", func(t *testing.T) {
		clt := fake.NewClient()
		c := NewController(clt, testNamespace)

		reconcileOnce(t, c)

		secretRV := getSecret(t, clt).ResourceVersion
		cmRV := getConfigMap(t, clt).ResourceVersion

		reconcileOnce(t, c)

		assert.Equal(t, secretRV, getSecret(t, clt).ResourceVersion)
		assert.Equal(t, cmRV, getConfigMap(t, clt).ResourceVersion)
	})

	t.Run("invalid CA is replaced", func(t *testing.T) {
		other, err := certificates.NewCA(time.Now())
		require.NoError(t, err)

		for name, key := range map[string][]byte{"garbage key": []byte("garbage"), "foreign key": other.KeyPEM, "missing key": nil} {
			t.Run(name, func(t *testing.T) {
				clt := fake.NewClient()
				c := NewController(clt, testNamespace)
				reconcileOnce(t, c)

				secret := getSecret(t, clt)
				oldCert := secret.Data[CACertKey]

				if key == nil {
					delete(secret.Data, CAKeyKey)
				} else {
					secret.Data[CAKeyKey] = key
				}

				require.NoError(t, clt.Update(t.Context(), secret))

				reconcileOnce(t, c)

				newSecret := getSecret(t, clt)
				assert.NotEqual(t, oldCert, newSecret.Data[CACertKey])

				_, err := loadCA(t.Context(), clt, testNamespace, time.Now())
				require.NoError(t, err)
				assert.Equal(t, string(newSecret.Data[CACertKey]), getConfigMap(t, clt).Data[CACertKey])
			})
		}
	})

	t.Run("expired CA is regenerated", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			clt := fake.NewClient()
			c := NewController(clt, testNamespace)
			reconcileOnce(t, c)

			old := getSecret(t, clt)
			ca, err := loadCA(t.Context(), clt, testNamespace, time.Now())
			require.NoError(t, err)

			time.Sleep(time.Until(ca.Cert.NotAfter) + time.Hour)

			reconcileOnce(t, c)

			newSecret := getSecret(t, clt)
			assert.NotEqual(t, old.Data[CACertKey], newSecret.Data[CACertKey])

			_, err = loadCA(t.Context(), clt, testNamespace, time.Now())
			require.NoError(t, err)
			assert.Equal(t, string(newSecret.Data[CACertKey]), getConfigMap(t, clt).Data[CACertKey])
		})
	})

	t.Run("deleted secret is recreated and bundle updated", func(t *testing.T) {
		clt := fake.NewClient()
		c := NewController(clt, testNamespace)
		reconcileOnce(t, c)

		oldCert := getSecret(t, clt).Data[CACertKey]

		require.NoError(t, clt.Delete(t.Context(), getSecret(t, clt)))
		reconcileOnce(t, c)

		newSecret := getSecret(t, clt)
		assert.NotEqual(t, oldCert, newSecret.Data[CACertKey])
		assert.Equal(t, string(newSecret.Data[CACertKey]), getConfigMap(t, clt).Data[CACertKey])
	})

	t.Run("tampered bundle is repaired and never contains the key", func(t *testing.T) {
		clt := fake.NewClient()
		c := NewController(clt, testNamespace)
		reconcileOnce(t, c)

		secretRV := getSecret(t, clt).ResourceVersion

		cm := getConfigMap(t, clt)
		cm.Data[CAKeyKey] = "secret"
		cm.Data[CACertKey] = "wrong"
		require.NoError(t, clt.Update(t.Context(), cm))

		reconcileOnce(t, c)

		cm = getConfigMap(t, clt)
		assert.Len(t, cm.Data, 1)
		assert.NotContains(t, cm.Data, CAKeyKey)
		assert.Equal(t, string(getSecret(t, clt).Data[CACertKey]), cm.Data[CACertKey])
		assert.Equal(t, secretRV, getSecret(t, clt).ResourceVersion)
	})

	t.Run("missing bundle is recreated", func(t *testing.T) {
		clt := fake.NewClient()
		c := NewController(clt, testNamespace)
		reconcileOnce(t, c)

		require.NoError(t, clt.Delete(t.Context(), getConfigMap(t, clt)))
		reconcileOnce(t, c)

		assert.Len(t, getConfigMap(t, clt).Data, 1)
	})

	t.Run("API error is returned", func(t *testing.T) {
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errAPI
			},
		})

		_, err := NewController(clt, testNamespace).Reconcile(t.Context(), reconcile.Request{})
		require.ErrorIs(t, err, errAPI)
	})
}
