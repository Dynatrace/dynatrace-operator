package kspm

import (
	"context"
	"testing"

	dtfake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm/consts"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTokenCreation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates secret if it does not exist", func(t *testing.T) {
		clt := fake.NewFakeClient()

		dk := createDynaKube(true)

		err := ensureKSPMToken(ctx, clt, clt, &dk)
		require.NoError(t, err)

		var secret v1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: dk.Name + "-" + consts.KSPMSecretKey, Namespace: dk.Namespace}, &secret)

		require.NotEmpty(t, secret)
		require.NoError(t, err)
	})

	t.Run("does not create secret if exists", func(t *testing.T) {
		dk := createDynaKube(true)
		objs := []client.Object{
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dk.Name + "-" + consts.KSPMSecretKey,
					Namespace: dk.Namespace,
				},
			},
		}
		clt := dtfake.NewClient(objs...)

		err := ensureKSPMToken(ctx, clt, clt, &dk)
		require.NoError(t, err)

		var secret v1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: dk.Name + "-" + consts.KSPMSecretKey, Namespace: dk.Namespace}, &secret)

		require.NotEmpty(t, secret)
		require.NoError(t, err)
	})
}
