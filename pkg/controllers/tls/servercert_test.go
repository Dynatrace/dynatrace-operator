// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	operatorNamespace = "dynatrace"
	ownerNamespace    = "consumer-ns"
	secretName        = "server-cert"
	componentName     = "test"
	serviceName       = "svc"
)

func newOwner() *dynakube.DynaKube {
	return &dynakube.DynaKube{Name: "owner", Namespace: ownerNamespace, UID: "owner-uid"}
}

func getServerSecret(t *testing.T, clt client.Client) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: secretName, Namespace: ownerNamespace}, secret))

	return secret
}

func setupIssuer(t *testing.T) (*Issuer, client.Client) {
	t.Helper()

	clt := fake.NewClient()
	_, err := NewController(clt, operatorNamespace).Reconcile(t.Context(), reconcile.Request{})
	require.NoError(t, err)

	return NewIssuer(clt, operatorNamespace, componentName, serviceName), clt
}

func TestServiceDNSNames(t *testing.T) {
	assert.ElementsMatch(t, []string{"svc", "svc.ns", "svc.ns.svc", "svc.ns.svc.cluster.local"}, serviceDNSNames("svc", "ns"))
	assert.Equal(t, "svc.ns", serviceDNSNames("svc", "ns")[0], "the first name is used as common name")
}

func TestEnsureServerCertSecret(t *testing.T) {
	t.Run("creates owned secret with hash annotation", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		secret := getServerSecret(t, clt)
		assert.Equal(t, corev1.SecretTypeTLS, secret.Type)
		assert.True(t, metav1.IsControlledBy(secret, owner))
		assert.Equal(t, hash, secret.Annotations[CertHashAnnotation])

		expected, err := computeHash(secret.Data)
		require.NoError(t, err)
		assert.Equal(t, expected, hash)
	})

	t.Run("second call is a no-op", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)

		rv := getServerSecret(t, clt).ResourceVersion

		hash2, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		assert.Equal(t, hash, hash2)
		assert.Equal(t, rv, getServerSecret(t, clt).ResourceVersion)
	})

	t.Run("missing annotation is restored without reissue", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)

		secret := getServerSecret(t, clt)
		crt := secret.Data[consts.TLSCrtDataName]
		delete(secret.Annotations, CertHashAnnotation)
		require.NoError(t, clt.Update(t.Context(), secret))

		hash2, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		assert.Equal(t, hash, hash2)
		assert.Equal(t, crt, getServerSecret(t, clt).Data[consts.TLSCrtDataName])
	})

	t.Run("reissues on SAN change", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)

		otherIssuer := NewIssuer(clt, operatorNamespace, componentName, "other-svc")

		hash2, err := otherIssuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		assert.NotEqual(t, hash, hash2)

		leaf, err := certificates.ParseCertificatePEM(getServerSecret(t, clt).Data[consts.TLSCrtDataName])
		require.NoError(t, err)
		assert.ElementsMatch(t, serviceDNSNames("other-svc", ownerNamespace), leaf.DNSNames)
	})

	t.Run("reissues on CA change", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)

		caSecret := &corev1.Secret{}
		require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: CASecretName, Namespace: operatorNamespace}, caSecret))
		require.NoError(t, clt.Delete(t.Context(), caSecret))
		_, err = NewController(clt, operatorNamespace).Reconcile(t.Context(), reconcile.Request{})
		require.NoError(t, err)

		hash2, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		assert.NotEqual(t, hash, hash2)

		require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: CASecretName, Namespace: operatorNamespace}, caSecret))
		assert.Equal(t, caSecret.Data["ca.crt"], getServerSecret(t, clt).Data["ca.crt"])
	})

	t.Run("reissues within the renewal threshold", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			issuer, _ := setupIssuer(t)
			owner := newOwner()

			hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
			require.NoError(t, err)

			time.Sleep(certificates.ServerCertDuration - certificates.ServerCertRenewalThreshold + time.Hour)

			hash2, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
			require.NoError(t, err)
			assert.NotEqual(t, hash, hash2)
		})
	})

	t.Run("does not reissue a leaf that is clamped to the CA validity", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			issuer, clt := setupIssuer(t)
			owner := newOwner()

			ca, err := loadCA(t.Context(), clt, operatorNamespace, time.Now())
			require.NoError(t, err)

			time.Sleep(time.Until(ca.Cert.NotAfter) - time.Hour)

			hash, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
			require.NoError(t, err)

			hash2, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
			require.NoError(t, err)
			assert.Equal(t, hash, hash2)
		})
	})

	t.Run("recreates secret of wrong type", func(t *testing.T) {
		issuer, clt := setupIssuer(t)
		owner := newOwner()

		require.NoError(t, clt.Create(t.Context(), &corev1.Secret{
			Name:      secretName,
			Namespace: ownerNamespace,
			Type:      corev1.SecretTypeOpaque,
		}))

		_, err := issuer.EnsureServerCertSecret(t.Context(), owner, secretName)
		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeTLS, getServerSecret(t, clt).Type)
	})

	t.Run("CA missing", func(t *testing.T) {
		clt := fake.NewClient()

		_, err := NewIssuer(clt, operatorNamespace, componentName, serviceName).EnsureServerCertSecret(t.Context(), newOwner(), secretName)
		require.Error(t, err)
		assert.True(t, IsCAUnavailable(err))
		require.ErrorIs(t, err, ErrCANotFound)
	})

	t.Run("CA invalid", func(t *testing.T) {
		issuer, clt := setupIssuer(t)

		caSecret := &corev1.Secret{}
		require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: CASecretName, Namespace: operatorNamespace}, caSecret))
		caSecret.Data["ca.key"] = []byte("garbage")
		require.NoError(t, clt.Update(t.Context(), caSecret))

		_, err := issuer.EnsureServerCertSecret(t.Context(), newOwner(), secretName)
		require.Error(t, err)
		assert.True(t, IsCAUnavailable(err))
	})

	t.Run("invalid arguments", func(t *testing.T) {
		issuer, _ := setupIssuer(t)

		_, err := issuer.EnsureServerCertSecret(t.Context(), nil, secretName)
		require.Error(t, err)

		_, err = issuer.EnsureServerCertSecret(t.Context(), &dynakube.DynaKube{}, secretName)
		require.Error(t, err)

		_, err = NewIssuer(fake.NewClient(), operatorNamespace, componentName, "").EnsureServerCertSecret(t.Context(), newOwner(), secretName)
		require.Error(t, err)
	})
}
