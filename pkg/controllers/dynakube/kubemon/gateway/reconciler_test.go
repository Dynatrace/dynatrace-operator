// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	tlsconsts "github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/gateway"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestKubemonDisabled(t *testing.T) {
	t.Run("no existing service/cert, reconcile completes without error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{},
		}
		err := gateway.NewReconciler(fake.NewClient(dk)).Reconcile(t.Context(), dk)
		require.NoError(t, err)
	})

	t.Run("existing service/cert are removed from the cluster", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{},
		}
		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.ServiceName(dk.Name),
				Namespace: dk.Namespace,
			},
		}
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.KubernetesMonitoring().GetAutoTLSSecretName(),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSvc, existingSecret)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		getErr := fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc)
		assert.True(t, k8serrors.IsNotFound(getErr), "service should have been removed from the cluster")

		secret := &corev1.Secret{}
		getErr = fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret)
		assert.True(t, k8serrors.IsNotFound(getErr), "secret should have been removed from the cluster")
	})
}

func TestKubemonEnabled(t *testing.T) {
	t.Run("no service/cert is created when KSPM is disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		fakeClient := fake.NewClient(dk)

		require.NoError(t, gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk))

		svc := &corev1.Service{}
		svcMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.True(t, svcMissing, "service should not have been created")

		secret := &corev1.Secret{}
		secretMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret))
		assert.True(t, secretMissing, "secret should not have been created")
	})

	t.Run("existing service/cert are removed when KSPM is disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.ServiceName(dk.Name),
				Namespace: dk.Namespace,
			},
		}
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.KubernetesMonitoring().GetAutoTLSSecretName(),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSvc, existingSecret)

		require.NoError(t, gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk))

		svc := &corev1.Service{}
		svcMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.True(t, svcMissing, "service should have been removed from the cluster")

		secret := &corev1.Secret{}
		secretMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret))
		assert.True(t, secretMissing, "secret should have been removed from the cluster")
	})

	t.Run("service has expected configuration", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
				KSPM:                 &kspm.Spec{},
			},
		}
		fakeClient := fake.NewClient(dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		helpers.AssertGolden(t, "testdata/service.yaml", svc)
	})

	t.Run("service creation failure is propagated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
				KSPM:                 &kspm.Spec{},
			},
		}
		createErr := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return createErr
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, createErr)
	})

	t.Run("automatic TLS secret is not created if custom secret is specified", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{
					TLSCertsRef: &kubemonapi.TLSCertsRef{
						SecretName: "custom-secret",
					},
				},
			},
		}
		fakeClient := fake.NewClient(dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		secret := &corev1.Secret{}
		secretMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret))
		assert.True(t, secretMissing, "secret should not have been created")
	})

	t.Run("existing automatic TLS secret is removed from the cluster if custom secret is specified", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{
					TLSCertsRef: &kubemonapi.TLSCertsRef{
						SecretName: "custom-secret",
					},
				},
				KSPM: &kspm.Spec{},
			},
		}
		customSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom-secret",
				Namespace: dk.Namespace,
			},
		}
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.KubernetesMonitoring().GetAutoTLSSecretName(),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSecret, customSecret)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		secret := &corev1.Secret{}
		secretMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret))
		assert.True(t, secretMissing, "secret should have been removed from the cluster")
	})

	t.Run("automatic TLS secret has expected configuration", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
				KSPM:                 &kspm.Spec{},
			},
		}
		fakeClient := fake.NewClient(dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		secret := &corev1.Secret{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetAutoTLSSecretName(), Namespace: dk.Namespace}, secret))

		assert.Contains(t, secret.Data, tlsconsts.TLSCrtDataName, "secret should contain "+tlsconsts.TLSCrtDataName+" field")
		assert.Contains(t, secret.Data, tlsconsts.TLSKeyDataName, "secret should contain "+tlsconsts.TLSKeyDataName+" field")
		assert.Contains(t, secret.Data, tlsconsts.TLSServerCrtDataName, "secret should contain "+tlsconsts.TLSServerCrtDataName+" field")
	})
}
