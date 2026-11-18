// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestDynaKubeReconcile(t *testing.T) {
	t.Run("valid CA sets condition true", func(t *testing.T) {
		clt := fake.NewClient()
		_, err := NewController(clt, testNamespace).Reconcile(t.Context(), reconcile.Request{})
		require.NoError(t, err)

		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace}}
		require.NoError(t, VerifyCA(t.Context(), clt, dk))

		cond := meta.FindStatusCondition(dk.Status.Conditions, ConditionType)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, ReasonCAReady, cond.Reason)
	})

	t.Run("missing CA sets condition false and creates nothing", func(t *testing.T) {
		clt := fake.NewClient()
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace}}

		require.NoError(t, VerifyCA(t.Context(), clt, dk))

		cond := meta.FindStatusCondition(dk.Status.Conditions, ConditionType)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, ReasonCANotFound, cond.Reason)

		err := clt.Get(t.Context(), types.NamespacedName{Name: CASecretName, Namespace: testNamespace}, &corev1.Secret{})
		require.Error(t, err)
		err = clt.Get(t.Context(), types.NamespacedName{Name: CABundleConfigMapName, Namespace: testNamespace}, &corev1.ConfigMap{})
		require.Error(t, err)
	})

	t.Run("invalid CA sets condition false without error and keeps secret", func(t *testing.T) {
		clt := fake.NewClient()
		_, err := NewController(clt, testNamespace).Reconcile(t.Context(), reconcile.Request{})
		require.NoError(t, err)

		secret := getSecret(t, clt)
		secret.Data[CAKeyKey] = []byte("garbage")
		require.NoError(t, clt.Update(t.Context(), secret))
		before := getSecret(t, clt)

		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace}}
		require.NoError(t, VerifyCA(t.Context(), clt, dk))

		cond := meta.FindStatusCondition(dk.Status.Conditions, ConditionType)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, ReasonCAInvalid, cond.Reason)

		after := getSecret(t, clt)
		assert.Equal(t, before.Data, after.Data)
		assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
	})

	t.Run("kube API error", func(t *testing.T) {
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errAPI
			},
		})
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace}}

		err := VerifyCA(t.Context(), clt, dk)
		require.Error(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, ConditionType)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, k8sconditions.KubeAPIErrorReason, cond.Reason)
	})
}

var errAPI = &apiError{}

type apiError struct{}

func (*apiError) Error() string { return "api error" }

func (*apiError) Status() metav1.Status { return metav1.Status{Status: metav1.StatusFailure} }
