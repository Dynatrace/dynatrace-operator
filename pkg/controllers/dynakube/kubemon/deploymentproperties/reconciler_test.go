// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/deploymentproperties"
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
	testNamespace              = "dynatrace"
	testDynakubeName           = "test-dk"
	testResourceAttributeKey   = "key"
	testResourceAttributeValue = "value"
	testDataValue              = "[resource_attributes]\n" + testResourceAttributeKey + " = " + testResourceAttributeValue + "\n"

	testResourceAttributeKey2   = "key2"
	testResourceAttributeValue2 = "value2"
	testDataValue2              = "[resource_attributes]\n" + testResourceAttributeKey2 + " = " + testResourceAttributeValue2 + "\n"

	testNoProxyValue     = "svc.cluster.local"
	testNoProxyWithComma = "svc.cluster.local,10.0.0.0/8"
)

func TestReconcile(t *testing.T) {
	t.Run("does not create configMap if KubeMon disabled", func(t *testing.T) {
		dk := newTestDynaKube(withoutKubernetesMonitoring())
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertDeploymentPropertiesSecretAbsent(t, clt, dk)
	})

	t.Run("does not create configMap if resourceAttributes property is empty", func(t *testing.T) {
		dk := newTestDynaKube(withoutResourceAttributes())
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertDeploymentPropertiesSecretAbsent(t, clt, dk)
	})

	t.Run("creates configMap", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		configMap := getDeploymentPropertiesSecret(t, clt, dk)
		assert.Equal(t, testDataValue, string(configMap.Data[agconsts.DeploymentPropertiesFileName]))
	})

	t.Run("updates configMap", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		configMap := getDeploymentPropertiesSecret(t, clt, dk)
		assert.Equal(t, testDataValue, string(configMap.Data[agconsts.DeploymentPropertiesFileName]))

		dk.Spec.ResourceAttributes = map[string]string{
			testResourceAttributeKey2: testResourceAttributeValue2,
		}

		require.NoError(t, r.Reconcile(t.Context(), dk))

		configMap = getDeploymentPropertiesSecret(t, clt, dk)
		assert.Equal(t, testDataValue2, string(configMap.Data[agconsts.DeploymentPropertiesFileName]))
	})

	t.Run("deletes configMap if resourceAttributes property is empty", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		configMap := getDeploymentPropertiesSecret(t, clt, dk)
		assert.Equal(t, testDataValue, string(configMap.Data[agconsts.DeploymentPropertiesFileName]))

		dk.Spec.ResourceAttributes = nil

		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertDeploymentPropertiesSecretAbsent(t, clt, dk)
	})

	t.Run("deletes configMap if KubeMon disabled", func(t *testing.T) {
		dk := newTestDynaKube()
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		configMap := getDeploymentPropertiesSecret(t, clt, dk)
		assert.Equal(t, testDataValue, string(configMap.Data[agconsts.DeploymentPropertiesFileName]))

		dk.Spec.KubernetesMonitoring = nil

		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertDeploymentPropertiesSecretAbsent(t, clt, dk)
	})

	t.Run("creates secret when only no-proxy is set (no resource attributes)", func(t *testing.T) {
		dk := newTestDynaKube(withoutResourceAttributes(), withNoProxy(testNoProxyValue))
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getDeploymentPropertiesSecret(t, clt, dk)
		expectedContent := agconsts.PropertiesClientInternalSection + "\n" + agconsts.PropertiesNoProxyFieldName + "=" + testNoProxyValue
		assert.Equal(t, expectedContent, string(secret.Data[agconsts.DeploymentPropertiesFileName]))
	})

	t.Run("creates secret with both resource attributes and no-proxy section", func(t *testing.T) {
		dk := newTestDynaKube(withNoProxy(testNoProxyValue))
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getDeploymentPropertiesSecret(t, clt, dk)
		expectedContent := testDataValue + agconsts.PropertiesClientInternalSection + "\n" + agconsts.PropertiesNoProxyFieldName + "=" + testNoProxyValue
		assert.Equal(t, expectedContent, string(secret.Data[agconsts.DeploymentPropertiesFileName]))
	})

	t.Run("replaces commas with pipes in no-proxy value", func(t *testing.T) {
		dk := newTestDynaKube(withoutResourceAttributes(), withNoProxy(testNoProxyWithComma))
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		secret := getDeploymentPropertiesSecret(t, clt, dk)
		expectedContent := agconsts.PropertiesClientInternalSection + "\n" + agconsts.PropertiesNoProxyFieldName + "=svc.cluster.local|10.0.0.0/8"
		assert.Equal(t, expectedContent, string(secret.Data[agconsts.DeploymentPropertiesFileName]))
	})

	t.Run("deletes secret when no-proxy removed and no resource attributes", func(t *testing.T) {
		dk := newTestDynaKube(withoutResourceAttributes(), withNoProxy(testNoProxyValue))
		clt := fake.NewClient(dk)

		r := deploymentproperties.NewReconciler(clt)
		require.NoError(t, r.Reconcile(t.Context(), dk))

		getDeploymentPropertiesSecret(t, clt, dk)

		delete(dk.Annotations, exp.NoProxyKey)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		assertDeploymentPropertiesSecretAbsent(t, clt, dk)
	})
}

func TestReconcileK8SAPIFailures(t *testing.T) {
	t.Run("returns error when secret create fails", func(t *testing.T) {
		dk := newTestDynaKube()
		errCreate := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errCreate
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk)

		r := deploymentproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errCreate)
	})

	t.Run("returns error when secret update fails", func(t *testing.T) {
		dk := newTestDynaKube()
		errUpdate := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errUpdate
				}

				return c.Update(ctx, obj, opts...)
			},
		}, dk, newExistingDeploymentPropertiesSecret(dk))

		r := deploymentproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errUpdate)
	})

	t.Run("returns error when secret deletion fails", func(t *testing.T) {
		dk := newTestDynaKube(withoutKubernetesMonitoring(), withoutResourceAttributes())
		errDelete := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return errDelete
			},
		}, dk, newExistingDeploymentPropertiesSecret(dk))

		r := deploymentproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errDelete)
	})

	t.Run("returns error when reading secret fails with a non-NotFound error", func(t *testing.T) {
		dk := newTestDynaKube()
		errGet := errors.New("kube api error")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok && key.Name == dk.KubernetesMonitoring().GetDeploymentPropertiesSecretName() {
					return errGet
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk)

		r := deploymentproperties.NewReconciler(clt)

		require.ErrorIs(t, r.Reconcile(t.Context(), dk), errGet)
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
			ResourceAttributes: map[string]string{
				testResourceAttributeKey: testResourceAttributeValue,
			},
		},
	}

	for _, mutate := range mutators {
		mutate(dk)
	}

	return dk
}

func withoutResourceAttributes() func(*dynakube.DynaKube) {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ResourceAttributes = nil
	}
}

func withoutKubernetesMonitoring() func(*dynakube.DynaKube) {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.KubernetesMonitoring = nil
	}
}

func withNoProxy(noProxy string) func(*dynakube.DynaKube) {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.Proxy = &value.Source{Value: "http://proxy:8080"}

		if dk.Annotations == nil {
			dk.Annotations = make(map[string]string)
		}

		dk.Annotations[exp.NoProxyKey] = noProxy
	}
}

func getDeploymentPropertiesSecret(t *testing.T, clt client.Client, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{
		Name:      dk.KubernetesMonitoring().GetDeploymentPropertiesSecretName(),
		Namespace: dk.Namespace,
	}, secret))

	return secret
}

func assertDeploymentPropertiesSecretAbsent(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	t.Helper()

	err := clt.Get(t.Context(), types.NamespacedName{
		Name:      dk.KubernetesMonitoring().GetDeploymentPropertiesSecretName(),
		Namespace: dk.Namespace,
	}, &corev1.Secret{})
	assert.True(t, k8serrors.IsNotFound(err), "expected NotFound, got %v", err)
}

func newExistingDeploymentPropertiesSecret(dk *dynakube.DynaKube) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetDeploymentPropertiesSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{agconsts.DeploymentPropertiesFileName: []byte("[resource_attributes]\na=b")},
	}
}
