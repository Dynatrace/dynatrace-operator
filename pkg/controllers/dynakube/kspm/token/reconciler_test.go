package token

import (
	"context"
	"errors"
	"testing"

	dtfake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestTokenCreation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates secret if it does not exist", func(t *testing.T) {
		clt := fake.NewFakeClient()

		dk := createDynaKube(true)

		err := ensureKSPMSecret(ctx, clt, clt, &dk)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace}, &secret)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		require.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), kspmConditionType))
		assert.Equal(t, conditions.SecretCreatedReason, meta.FindStatusCondition(*dk.Conditions(), kspmConditionType).Reason)
		assert.NotEmpty(t, dk.KSPM().TokenSecretHash)
	})

	t.Run("unexpected error -> return error", func(t *testing.T) {
		clt := createFailK8sClient(t)

		dk := createDynaKube(true)

		err := ensureKSPMSecret(ctx, clt, clt, &dk)
		require.Error(t, err)
		assert.Equal(t, conditions.KubeApiErrorReason, meta.FindStatusCondition(*dk.Conditions(), kspmConditionType).Reason)
	})

	t.Run("removes secret if exists", func(t *testing.T) {
		dk := createDynaKube(false)
		dk.KSPM().TokenSecretHash = "something"
		conditions.SetSecretCreated(dk.Conditions(), kspmConditionType, dk.KSPM().GetTokenSecretName())

		objs := []client.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dk.KSPM().GetTokenSecretName(),
					Namespace: dk.Namespace,
				},
			},
		}
		clt := dtfake.NewClient(objs...)

		reconciler := &Reconciler{
			client:    clt,
			apiReader: clt,
			dk:        &dk,
		}

		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: dk.KSPM().GetTokenSecretName(), Namespace: dk.Namespace}, &secret)

		require.Error(t, err)
		assert.Empty(t, secret)
		assert.Empty(t, dk.KSPM().TokenSecretHash)
	})
}

func createFailK8sClient(t *testing.T) client.Client {
	t.Helper()

	boomClient := dtfake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}

func createDynaKube(kspmEnabled bool) dynakube.DynaKube {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{},
	}

	if kspmEnabled {
		dk.KSPM().Spec = &kspm.Spec{}
	} else {
		dk.KSPM().Spec = nil
	}

	return dk
}
