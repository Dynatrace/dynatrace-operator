// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package customproperties_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/customproperties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace          = "dynatrace"
	testDynakubeName       = "test-dk"
	testInlineValue        = "[section]\nkey=inline"
	testReferencedValue    = "[section]\nkey=from-secret"
	testReferencedSecret   = "user-custom-properties"
	testReferencedRotation = "[section]\nkey=updated"
)

func TestReconcile(t *testing.T) {
	t.Run("creates secret from inline value", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getCustomPropertiesSecret(t, clt, dk)
		assert.Equal(t, []byte(testInlineValue), secret.Data[customproperties.DataKey])
	})

	t.Run("creates secret from referenced Secret", func(t *testing.T) {
		dk := newTestDynaKube(withValueFrom(testReferencedSecret))
		clt := fake.NewClient(dk, newReferencedSecret(testReferencedValue))

		r := customproperties.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getCustomPropertiesSecret(t, clt, dk)
		assert.Equal(t, []byte(testReferencedValue), secret.Data[customproperties.DataKey])
	})

	t.Run("updates existing secret when inline value changes", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		dk.Spec.KubernetesMonitoring.CustomProperties.Value = testReferencedRotation
		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getCustomPropertiesSecret(t, clt, dk)
		assert.Equal(t, []byte(testReferencedRotation), secret.Data[customproperties.DataKey])
	})

	t.Run("cleans up secret when CustomProperties becomes nil but kubemon stays enabled", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		dk.Spec.KubernetesMonitoring.CustomProperties = nil
		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertSecretAbsent(t, clt, dk)
	})

	t.Run("cleans up secret when kubemon is disabled", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		dk.Spec.KubernetesMonitoring = nil
		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertSecretAbsent(t, clt, dk)
	})

	t.Run("errors when referenced Secret has empty customProperties key", func(t *testing.T) {
		dk := newTestDynaKube(withValueFrom(testReferencedSecret))
		empty := newReferencedSecret("")
		clt := fake.NewClient(dk, empty, newExistingCustomPropertiesSecret(dk, testInlineValue))

		r := customproperties.NewReconciler(clt)

		require.Error(t, r.Reconcile(t.Context(), dk))
	})

	t.Run("no-op when nothing to clean and CustomProperties is nil", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertSecretAbsent(t, clt, dk)
	})

	t.Run("inline value wins over ValueFrom when both are set", func(t *testing.T) {
		dk := newTestDynaKube(func(dk *dynakube.DynaKube) {
			dk.Spec.KubernetesMonitoring.CustomProperties = &value.Source{
				Value:     testInlineValue,
				ValueFrom: testReferencedSecret,
			}
		})
		// ValueFrom secret intentionally absent — reconcile must not touch it.
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getCustomPropertiesSecret(t, clt, dk)
		assert.Equal(t, []byte(testInlineValue), secret.Data[customproperties.DataKey])
	})
}

// TestReconcilePreconditionErrors covers failures resolving Spec.CustomProperties before any write.
func TestReconcilePreconditionErrors(t *testing.T) {
	t.Run("returns error when referenced Secret does not exist", func(t *testing.T) {
		dk := newTestDynaKube(withValueFrom(testReferencedSecret))
		clt := fake.NewClient(dk)

		r := customproperties.NewReconciler(clt)

		err := r.Reconcile(t.Context(), dk)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "expected NotFound, got %v", err)
	})

	t.Run("returns error when reading referenced Secret fails with a non-NotFound error", func(t *testing.T) {
		dk := newTestDynaKube(withValueFrom(testReferencedSecret))
		errGet := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok && key.Name == testReferencedSecret {
					return errGet
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk)

		r := customproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errGet)
	})
}

// TestReconcileWriteFailures covers Kubernetes API failures on the create/update path.
func TestReconcileWriteFailures(t *testing.T) {
	t.Run("returns error when secret create fails", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		errCreate := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errCreate
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk)

		r := customproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errCreate)
	})

	t.Run("returns error when secret update fails", func(t *testing.T) {
		dk := newTestDynaKube(withInline(testInlineValue))
		errUpdate := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errUpdate
				}

				return c.Update(ctx, obj, opts...)
			},
		}, dk, newExistingCustomPropertiesSecret(dk, "[section]\nkey=stale"))

		r := customproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errUpdate)
	})
}

// TestReconcileCleanupFailures covers the delete failure on the cleanup path.
func TestReconcileCleanupFailures(t *testing.T) {
	t.Run("returns error when secret deletion fails", func(t *testing.T) {
		dk := newTestDynaKube()
		errDelete := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return errDelete
			},
		}, dk, newExistingCustomPropertiesSecret(dk, testInlineValue))

		r := customproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errDelete)
	})
}

func newTestDynaKube(mutators ...func(*dynakube.DynaKube)) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:               "https://tenant.live.dynatrace.com/api",
			KubernetesMonitoring: &kubemonapi.Spec{},
		},
	}

	for _, mutate := range mutators {
		mutate(dk)
	}

	return dk
}

func withInline(v string) func(*dynakube.DynaKube) {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.KubernetesMonitoring.CustomProperties = &value.Source{Value: v}
	}
}

func withValueFrom(name string) func(*dynakube.DynaKube) {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.KubernetesMonitoring.CustomProperties = &value.Source{ValueFrom: name}
	}
}

func newReferencedSecret(v string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testReferencedSecret,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{customproperties.DataKey: []byte(v)},
	}
}

func newExistingCustomPropertiesSecret(dk *dynakube.DynaKube, v string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{customproperties.DataKey: []byte(v)},
	}
}

func getCustomPropertiesSecret(t *testing.T, clt client.Client, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{
		Name:      dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
		Namespace: dk.Namespace,
	}, secret))

	return secret
}

func assertSecretAbsent(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	t.Helper()

	err := clt.Get(t.Context(), types.NamespacedName{
		Name:      dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
		Namespace: dk.Namespace,
	}, &corev1.Secret{})
	assert.True(t, k8serrors.IsNotFound(err), "expected NotFound, got %v", err)
}
